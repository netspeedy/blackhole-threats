# Release And Publishing

This guide explains how changes move from `main` to release candidates, stable
tags, GitHub Releases, GHCR images, and the signed APT repository.

See also:

- [Documentation Index](README.md)
- [How it works](https://netspeedy.github.io/blackhole-threats/how-it-works/), on the docs site

## Release Model

The repository is designed so that `main` is release-candidate-ready.

That means a successful validation run for a push to `main` can trigger the
release-candidate path automatically:

1. compute the next semantic version
2. create a prerelease tag `v<major>.<minor>.<patch>-rc.<n>`
3. validate prerelease asset and container publication
4. if the release only holds automated updates, promote the candidate to the
   stable tag on the same commit and wait for the stable publish workflows,
   including the APT repository

## Workflow Map

### Build and Validate

Purpose:

- formatting checks
- vet and tests
- native binary build
- binary smoke tests
- Debian package validation
- signed APT repository layout validation
- docs site build and validation (`make website-build`, then `make website-check`)

This is the gate that must pass before automatic release automation acts on a
push to `main`.

### Automated Release Candidate

Trigger:

- successful `Build and Validate` completion for a push to `main`
- manual `workflow_dispatch`

Behavior:

- checks whether `HEAD` already carries a release tag
- runs `scripts/next-release.sh`
- skips tagging if no release-bearing commits are queued
- updates `debian/changelog` for the next stable Debian package version when
  needed, pushes that changelog commit, and waits for the follow-up validation
  run before tagging
- creates the next `-rc` tag when a release is required
- dispatches `container-image.yml` and `release-assets.yml` from the current
  default branch with the target tag passed explicitly
- waits for those prerelease publish paths to succeed
- checks with `scripts/automated-only.sh` whether every release-bearing commit
  since the last stable release came from Dependabot, Renovate or this
  repository's workflows, and stops at the candidate if any came from a person
- if so, tags the same commit as the stable release, which needs
  `RELEASE_AUTOMATION_TOKEN` so the tag starts its publish workflows, and waits
  for `release-assets.yml`, `container-image.yml` and
  `publish-apt-repository.yml` to finish for it

Recovery path:

- if a tagged publish ever needs to be replayed, dispatch the publish workflow
  from the current default branch and set `release_ref` to the existing tag
  rather than re-running an old tag-pinned workflow definition

Important consequences:

- stable tags for automated updates are created by the release automation once
  the candidate's publish path passes; a release that includes a change a
  person merged is promoted with the `Promote Release Candidate` workflow, which
  also finishes a run that failed partway
- release candidates and their matching stable releases point at the same
  commit, so the checked-in changelog records the stable package version while
  RC package builds rewrite their build-local changelog version to `~rc.N`

### Release Drafter

Purpose:

- apply release-note labels on pull requests
- maintain a curated draft release for queued release-bearing work
- remove stale automated drafts when no release-bearing changes remain

This keeps the GitHub Releases page aligned with the actual release gate.

### Release Assets

Trigger:

- trusted `v*` tags
- manual `workflow_dispatch` with `release_ref=<tag>` for publish recovery

Published outputs:

- Linux binaries for `amd64`, `arm64`, and `arm`
- Debian runtime packages for `amd64`, `arm64`, and `armhf`
- Debian source package artifacts
- `sha256sums.txt`
- curated GitHub Release notes and assets
- for the first release of a new line (`vX.Y.0`), the outgoing line's docs as
  `blackhole-threats-docs-<line>.tar.gz`, attached to that line's last release
  (see [Versioned docs](#versioned-docs))

### Container Image

Triggers:

- pull requests for validation-only builds
- trusted `v*` tags
- manual `workflow_dispatch` for validation, or with `release_ref=<tag>` for publish recovery

Published channels:

- `ghcr.io/netspeedy/blackhole-threats:rc` plus full `v*-rc.*` tags for release candidates
- stable semver tags and `latest` for promoted stable releases

### Publish Signed Debian Repository

Trigger:

- trusted stable `v*` tags only
- manual `workflow_dispatch` with `release_ref=<stable-tag>` for recovery

Outputs:

- Debian binary packages
- Debian source indexes
- signed `Release.gpg` and `InRelease`
- public archive key exports
- the docs site, built with the release's tag and the archive key's fingerprint,
  with the archived release lines beside it, overlaid onto the repository root

Stable releases publish the machine-readable APT repository and the human-facing
docs site together.

### Deploy Pages Site

Trigger:

- pushes to `main`
- successful `Publish Signed Debian Repository` completion
- successful `Release Assets` completion for a stable tag, which is when an
  outgoing line's docs archive is attached

Purpose:

- refresh the website without cutting a release
- reuse the latest published repository snapshot
- avoid rebuilding unsigned APT metadata for website-only changes
- republish the default-branch Pages artifact after stable APT publication so
  the public docs site follows the latest signed snapshot
- publish the archived docs of older release lines beside the latest

## Website

The site at https://netspeedy.github.io/blackhole-threats/ is the blackhole-threats
documentation, built with Astro and Starlight from `website/`. The pages are
Markdown and MDX in `website/src/content/docs`. Run it locally with live reload:

```bash
make website-install
make website-dev
```

Build it, validate the output, drive it in a browser, and take desktop and phone
screenshots in both themes:

```bash
make website-build
make website-check
make website-test
make website-capture
```

`website-test` runs `website/scripts/browser-test.mjs` in headless Chrome. It
builds the site twice into `website/.test/`, as an archived line and as the
latest with that line listed, unpacks the archive the way a Pages build would,
and checks the theme controls, the header, each tab's sidebar, the version menu
(including a long list of lines) and the archived notice. It reads the site
root, the tabs and the pages from the build, and exits non-zero if a check
fails.

`website-check` validates `website/dist`, so it runs after the build. Captures
are written to `website/.captures/`; the first one downloads the Chrome build
Puppeteer pins, since installing the site's dependencies never does. Search only
works in a build, because Pagefind indexes the finished pages.

The build always starts with a fresh content cache (`astro build --force`, which
logs "data store cleared"). Astro notices changes to its config, but not edits
to the Markdown plugins in `website/src/lib`, so without it a plugin change
could leave pages rendered by the old code.

The site shares the Pages root with the APT repository. Both Pages workflows
write the stable release's tag and the archive key's fingerprint to a JSON file
before they build, and pass it to the site in `SITE_METADATA_FILE`, so the
published pages name the release and show the key's fingerprint. Publish Signed
Debian Repository takes them from the tag it's publishing and the key it signed
with; Deploy Pages Site takes them from the restored APT snapshot. Local and
pull request builds fall back to the newest local tag and leave the fingerprint
out. The same JSON is published as `website-metadata.json`. Links from the pages
to the APT files are absolute URLs, because those files aren't part of the
build and `website-check` only follows links into it.

### Versioned docs

The docs are versioned by release line. The latest line is built at the site
root, and each older line is served under its own path, such as
`https://netspeedy.github.io/blackhole-threats/v1.1/`. The version menu in the
header lists them, and an older line's pages open with a notice that links to
the same page in the latest docs.

Only the latest line is kept in the repository. When the first release of a new
line is published, such as `v1.2.0`, the Release Assets workflow finds the
commit just before the new line started (`scripts/docs-archive-commit.sh`,
which uses the same `feat:` and breaking-change rules as `next-release.sh`),
builds the site from it with `SITE_ARCHIVE` set to the old line, and attaches
the result to the old line's last release as
`blackhole-threats-docs-<line>.tar.gz`. The archived pages show that release
and the archive key's current fingerprint. The step that runs `npm ci` and the
build on the older commit has no token in its environment; only the separate
Publish Archived Docs job can write to the release. A failed archive doesn't
fail the release, which the release automation waits on: the job reports it in
its summary and leaves the line unpublished.

Both Pages workflows deploy the whole site, so both run
`scripts/download-docs-archives.sh`, which unpacks every older line's archive
into the build before `make website-check` and lists the lines in the site
metadata's `docs_versions`. They end up in `versions.json` at the root, which
archived lines read when they load, so their menus include lines released after
them. Publish Signed Debian Repository takes the script from the workflow's own
commit, since a release published again by hand checks out an older tag, and
carries on without the archived lines if it fails; Deploy Pages Site runs after
it and after Release Assets, so whichever finishes last publishes the new
archive.

v1.1 is the first line that gets an archive, when v1.2.0 ships. A line whose
last commit predates the Starlight site is skipped, because it can't build under
a line's own path.

To build an archived line by hand, for example to replace a lost archive, check
out the commit the script prints and build it with the line set. Without
`SITE_METADATA_FILE` the pages leave out the APT fingerprint; the docs-archive
job in `.github/workflows/release-assets.yml` shows how it writes that file.

```bash
git checkout "$(bash scripts/docs-archive-commit.sh v1.1.7 v1.2.0)"
SITE_ARCHIVE=v1.1 make website-build
SITE_ARCHIVE=v1.1 make website-check
tar -C website/dist -czf blackhole-threats-docs-v1.1.tar.gz .
gh release upload v1.1.7 blackhole-threats-docs-v1.1.tar.gz --clobber
```

Then run Deploy Pages Site to publish it.

`TestCLIReferenceCoversEveryFlag` in `cmd/blackhole-threats` fails when a flag
is added to or removed from the CLI without updating
`website/src/content/docs/reference/cli.mdx`.

The favicon, Apple touch icon, `favicon.ico`, the mark and the share card in
`website/` come from `make readme-hero`, along with the README banner.

## Version Bump Rules

The semantic bump comes from commit history inspected by
`scripts/next-release.sh`.

Current rules:

- major:
  - `BREAKING CHANGE:`
  - `type!:` style conventional subjects
- minor:
  - `feat:`
- patch:
  - `fix:`
  - `perf:`
  - `revert:`
  - `container:`
  - `build:`
  - `deps:`
  - `packaging:`

These types do not cut a release by default:

- `docs:`
- `ci:`
- `chore:`
- `test:`
- `refactor:`

## Automation Update Path

Dependabot updates and scheduled build/runtime pin refreshes are part of the
normal automation path.

Current behavior:

- Dependabot PRs and scheduled build/runtime pin refresh PRs run through
  `Build and Validate`
- the pin refresh PR is opened with `GITHUB_TOKEN`, so its `pull_request`
  workflows don't start; the refresh dispatches `Build and Validate` and a
  validation-only `Container Image` build on its branch instead
- green PRs can be merged directly by the dedicated automation merge workflow;
  Dependabot PRs merge as soon as `Build and Validate` completes, and a daily
  05:00 UTC sweep merges the rest, including the pin refresh, once both
  `Build and Validate` and `Container Image` have passed on the PR head
- when GitHub cannot merge immediately but the PR is otherwise eligible, the
  workflow enables GitHub auto-merge so the PR can land as soon as the
  remaining repository requirements are satisfied
- the original PR title is preserved as the squash commit subject
- `deps:` and `container:` updates remain release-bearing after merge; the pin
  refresh is titled `container:` too, since everything it changes is in the
  published image
- `ci:` and `chore:` updates remain non-release-bearing; the website's npm
  updates are titled `chore:`, since the site isn't part of the package
- patch, minor, and major version updates are all eligible for merge after
  validation
- merges use the `AUTOMATION_AUTOMERGE_TOKEN` organisation secret. Without it
  the workflow falls back to the repo-scoped `GITHUB_TOKEN`, whose merges start
  no push workflows, so `main` isn't validated and nothing releases, and which
  can't merge PRs that change workflow files
- the release workflows publish directly to the tagged GitHub Release rather
  than creating a temporary draft shell, and they clean up stale orphan
  `untagged-*` drafts for the same tag before publishing
- release tags are pushed with `RELEASE_AUTOMATION_TOKEN`, the organisation
  secret, because a tag pushed with the repo-scoped `GITHUB_TOKEN` doesn't start
  the workflows that publish it, and a release-bearing commit under
  `.github/workflows/` needs the workflow permission that token carries

Operationally, this means passing dependency updates flow from PR to a
published stable release, APT package and `latest` image without manual
intervention, while a change a person merged gets a release candidate and waits
for a maintainer to promote it.

## Trust Boundaries

The release automation is deliberately strict because this is a public
repository.

Key constraints:

- release and publish tags must point to commits already contained in `main`
- pull requests do not receive publish or signing credentials
- write permissions are granted only to the jobs that need them
- APT signing secrets live in the protected `apt-repository` environment
- stable APT publication is restricted to stable tags, not `main`

## When Manual Intervention Is Still Needed

Automation is the default, but a maintainer still steps in when:

- `Build and Validate` fails on `main`
- a release-bearing commit updates `.github/workflows/` and
  `RELEASE_AUTOMATION_TOKEN` is not configured, so GitHub blocks the automated
  RC tag push
- prerelease asset publication fails
- container publication fails
- APT signing or Pages publication fails
- a release candidate includes a change a person merged and is ready to be
  promoted with the `Promote Release Candidate` workflow, or an automatic
  promotion failed partway and needs finishing
- a dependency update stays red and should not merge automatically

## What Does Not Create A Release

By default, these changes should not create a new runtime release on their own:

- documentation-only work
- README-only edits
- architecture-only edits
- website-only changes
- Pages-only refreshes
- CI-only changes using non-release-bearing commit types

Those changes may still refresh the docs site from `main`, but they should
not generate a new release tag unless they intentionally change shipped runtime,
container, packaging, or release-pipeline behavior.

## Troubleshooting Releases

These are the checks for release, automation and publishing problems; problems
running the daemon are covered on the
[docs site](https://netspeedy.github.io/blackhole-threats/guides/troubleshooting/).
A stable release isn't healthy until Release Assets, Container Image and Publish
Signed Debian Repository have all succeeded for the same tag, and the signed APT
repository and the GitHub release assets agree.

### Release Draft Looks Wrong Or Empty

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

### An Automation PR Did Not Merge Automatically

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
- confirm the `AUTOMATION_AUTOMERGE_TOKEN` organisation secret is still shared
  with this repository. The sweep merges with it so the merge starts `main`'s
  workflows, and only it can merge PRs that change `.github/workflows/`; a merge
  made with `GITHUB_TOKEN` starts nothing, so no release follows

If the PR is red, that is the intended stopping point for the automation path.

### Release Page Shows Both A Draft And A Published Release For The Same Tag

Checks:

- confirm the published tagged release succeeded in `Release Assets`
- look for a stale orphan draft whose URL ends with `untagged-*` but whose
  `tag_name` still matches the real release tag
- delete only the orphan draft release object by ID; do not delete the real
  published release for the same tag
- update to the current `Release Assets` workflow if the release was created by
  older automation, because the current flow publishes directly to the tagged
  release and removes stale orphan drafts before publishing

### Signed APT Repository Did Not Update

Checks:

- confirm the release reached a stable `v*` tag rather than stopping at `-rc`;
  one that includes a change a person merged stops at the candidate until
  someone runs `Promote Release Candidate`
- confirm `Publish Signed Debian Repository` succeeded
- confirm the `apt-repository` environment secrets are present
- confirm GitHub Pages is enabled with GitHub Actions as the publishing source
- if the original stable-tag run used stale workflow logic, dispatch `Publish
  Signed Debian Repository` from the current default branch with
  `release_ref=<stable-tag>`
- confirm the `RELEASE_AUTOMATION_TOKEN` organisation secret is still shared
  with this repository; the stable tag is pushed with it, because a tag pushed
  with `GITHUB_TOKEN` doesn't start the publish workflows

Useful things to verify in the published site:

- `dists/stable/Release`
- `dists/stable/InRelease`
- `dists/stable/main/binary-amd64/Packages`
- `dists/stable/main/source/Sources`
- `blackhole-threats-archive-keyring.gpg`
