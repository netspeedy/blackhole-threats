# Troubleshooting

This guide collects the higher-signal troubleshooting steps for runtime,
configuration, feed ingestion, packaging, and release automation issues.

See also:

- [Operations Guide](operations.md)
- [Feed Behavior](feed-behavior.md)
- [Release And Publishing](release-and-publishing.md)

## Start Here

Before diving into a specific symptom, check:

1. The last startup log lines.
2. The most recent `Refresh completed` line.
3. Whether the issue is a config problem, a feed problem, or a BGP/session
   problem.

The fastest safe validation commands are:

```bash
./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml -check-config
./dist/blackhole-threats -conf /path/to/blackhole-threats.yaml -once
```

## Configuration Fails To Load

Typical logs:

```text
Failed to load configuration
Invalid configuration
Invalid feed configuration
```

Checks:

- confirm the `-conf` path exists
- confirm the service user can read the file
- verify there are no unknown YAML keys
- verify `gobgp.global.config.as` is non-zero
- verify `gobgp.global.config.routerid` is a valid IPv4 address
- verify every feed URL is non-empty and uses a supported scheme

Packaged install:

```bash
sudo /usr/sbin/blackhole-threats -conf /etc/blackhole-threats.yaml -check-config
```

## Feed Errors Keep Repeating

Typical log:

```text
Failed to read threat feed
```

Checks:

- confirm the URL is still reachable
- confirm HTTP endpoints return `200 OK`
- confirm local file paths still exist
- confirm JSON feeds still emit valid JSON
- check for provider-side format changes

Operational note:

If a source in a community fails, the daemon carries forward the last
known-good routes for that community instead of withdrawing them immediately.

That is safer than withdrawing on uncertainty, but it also means repeated feed
errors can leave you advertising stale data longer than intended.

## Manual Refresh Did Not Apply My YAML Edit

This is expected.

`SIGUSR1` and `systemctl reload blackhole-threats` trigger an immediate feed
refresh only. They do not reload the YAML configuration or command-line flags.

Use:

```bash
sudo systemctl restart blackhole-threats
```

or the equivalent process/container restart when the YAML changed.

## Routes Are Not Appearing

Checks:

- confirm the BGP session is actually established
- verify the router ID and ASN are correct
- if using non-standard BGP ports, verify `gobgp.global.config.port` for the
  local listener and `gobgp.neighbors[].config.port` for the remote peer
- confirm the downstream router policy matches the selected communities
- run with `-log-level debug` or `-debug`
- verify the feeds produced usable prefixes
- confirm route count is not being reduced unexpectedly by summarisation

Useful logs to inspect:

- `Prepared community routes`
- `Refresh completed`
- `announced`
- `withdrawn`
- `active_routes`

## Unexpectedly Low Route Count

Possible causes:

- the source content changed upstream
- multiple feeds now overlap heavily
- summarisation merged sibling prefixes
- broader prefixes now contain more specific entries
- an entire community fell back to carried-forward routes after a feed failure

Checks:

- compare current feed contents with earlier snapshots
- inspect `prefixes_read` versus `prefixes_summarized`
- look for `Keeping previous community routes after feed errors`

## Container Starts But No Config Appears

Checks:

- confirm the `/config` mount exists
- confirm the mount is writable
- confirm you did not override `BLACKHOLE_THREATS_CONF` to a missing path

Useful commands:

```bash
docker logs blackhole-threats
docker exec -it blackhole-threats ls -l /config
```

## Debian Service Does Not Stay Running

Checks:

- inspect `systemctl status blackhole-threats`
- inspect `journalctl -u blackhole-threats`
- confirm `/etc/blackhole-threats.yaml` exists and is readable
- review `/etc/default/blackhole-threats`
- confirm any extra flags in `EXTRA_OPTS` are valid

Useful commands:

```bash
sudo systemctl status blackhole-threats
sudo journalctl -u blackhole-threats -n 100 --no-pager
sudo cat /etc/default/blackhole-threats
```

## Release Draft Looks Wrong Or Empty

Current behavior:

- the release gate should prevent new drafts when no release-bearing commits are queued
- stale automated drafts are deleted by the Release Drafter workflow

If you still see an empty draft:

- confirm the latest `Release Drafter` workflow run succeeded
- confirm the workflow checked the current repository tags before evaluating
  `scripts/next-release.sh`
- inspect whether the commit history since the last stable tag contains only
  non-release-bearing commit types
- confirm the stale draft body matches the automated empty-draft pattern rather
  than a maintainer-created draft

If you still see a non-empty automated draft even though only non-release-
bearing commits landed:

- confirm the latest `Release Drafter` workflow fetched tags before running the
  release gate
- confirm the draft body matches the automated `## Included Changes` template
  rather than a maintainer-created draft

If you see a non-empty draft for a tag that should already be published:

- confirm the matching stable `Release Assets` run succeeded
- if the failed run came from an older tag snapshot, dispatch `Release Assets`
  from the current default branch with `release_ref=<tag>` so the recovery uses
  the latest workflow logic
- check `Container Image` and `Publish Signed Debian Repository` for the same
  tag too; a stable release is not fully healthy until all three publish paths
  agree

## An Automation PR Did Not Merge Automatically

Checks:

- confirm `Build and Validate` succeeded on the PR head revision
- for a PR merged by the daily sweep, such as the build/runtime pin refresh,
  confirm `Container Image` succeeded on the head revision too; the refresh
  dispatches both, and `Dependabot Merge` can be dispatched by hand to sweep
  again without waiting for 05:00 UTC
- confirm the PR is still open and not in draft state
- confirm it is authored by `dependabot[bot]` or `github-actions[bot]`
- confirm there are no merge conflicts
- confirm branch protection still allows squash merge after checks pass
- if GitHub says to add `--auto`, confirm the workflow enabled auto-merge and
  let GitHub finish the merge after the remaining requirements settle
- if the PR changes files under `.github/workflows/`, confirm the optional
  `AUTOMATION_AUTOMERGE_TOKEN` secret exists with repository contents, pull
  requests, and workflows write access; the default `GITHUB_TOKEN` cannot
  merge workflow-file PRs under GitHub's workflow-permission rules

If the PR is red, that is the intended stopping point for the automation path.

## Release Page Shows Both A Draft And A Published Release For The Same Tag

Checks:

- confirm the published tagged release succeeded in `Release Assets`
- look for a stale orphan draft whose URL ends with `untagged-*` but whose
  `tag_name` still matches the real release tag
- delete only the orphan draft release object by ID; do not delete the real
  published release for the same tag
- update to the current `Release Assets` workflow if the release was created by
  older automation, because the current flow publishes directly to the tagged
  release and removes stale orphan drafts before publishing

## Signed APT Repository Did Not Update

Checks:

- confirm the release reached a stable `v*` tag rather than stopping at `-rc`
- confirm `Publish Signed Debian Repository` succeeded
- confirm the `apt-repository` environment secrets are present
- confirm GitHub Pages is enabled with GitHub Actions as the publishing source
- if the original stable-tag run used stale workflow logic, dispatch `Publish
  Signed Debian Repository` from the current default branch with
  `release_ref=<stable-tag>`
- if the release-bearing commit changed `.github/workflows/`, confirm the
  optional `RELEASE_AUTOMATION_TOKEN` secret exists with repository contents
  and workflows write access; the default `GITHUB_TOKEN` can be blocked from
  pushing the stable tag in that case

Useful things to verify in the published site:

- `dists/stable/Release`
- `dists/stable/InRelease`
- `dists/stable/main/binary-amd64/Packages`
- `dists/stable/main/source/Sources`
- `blackhole-threats-archive-keyring.gpg`

## When To Treat It As An Incident

Escalate promptly if:

- the daemon stops refreshing entirely
- the same community keeps falling back for multiple refresh intervals
- route counts swing sharply after a feed or policy change
- stable releases are tagging but publish workflows are failing
- the signed APT repository and GitHub release assets diverge from the same tag
