# Release And Publishing

This guide explains how changes move from `main` to release candidates, stable
tags, GitHub Releases, GHCR images, and the signed APT repository.

See also:

- [Architecture](architecture.md)
- [Documentation Index](README.md)

## Release Model

The repository is designed so that `main` is release-candidate-ready.

That means a successful validation run for a push to `main` can trigger the
release-candidate path automatically:

1. compute the next semantic version
2. create a prerelease tag `v<major>.<minor>.<patch>-rc.<n>`
3. validate prerelease asset and container publication
4. promote the candidate to the stable tag on the same commit and wait for the
   stable publish workflows, including the APT repository

## Workflow Map

### Build and Validate

Purpose:

- formatting checks
- vet and tests
- native binary build
- binary smoke tests
- Debian package validation
- signed APT repository layout validation
- website validation

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
- tags the same commit as the stable release, which needs
  `RELEASE_AUTOMATION_TOKEN` so the tag starts its publish workflows, and waits
  for `release-assets.yml`, `container-image.yml` and
  `publish-apt-repository.yml` to finish for it

Recovery path:

- if a tagged publish ever needs to be replayed, dispatch the publish workflow
  from the current default branch and set `release_ref` to the existing tag
  rather than re-running an old tag-pinned workflow definition

Important consequences:

- stable tags are created by the release automation once the candidate's
  publish path passes; the `Promote Release Candidate` workflow is only for
  recovering from a run that failed partway
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
- GitHub Pages landing site overlaid onto the repository root

Stable releases publish the machine-readable APT repository and the human-facing
landing page together.

### Deploy Pages Site

Trigger:

- pushes to `main`
- successful `Publish Signed Debian Repository` completion

Purpose:

- refresh the website without cutting a release
- reuse the latest published repository snapshot
- avoid rebuilding unsigned APT metadata for website-only changes
- republish the default-branch Pages artifact after stable APT publication so
  the public landing site follows the latest signed snapshot

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
- `deps:` and `container:` updates remain release-bearing after merge
- `ci:` and `chore:` updates remain non-release-bearing; the website's npm
  updates are titled `chore:`, since the site isn't part of the package
- patch, minor, and major version updates are all eligible for merge after
  validation
- the repo-scoped `GITHUB_TOKEN` is the default merge credential, and an
  optional `AUTOMATION_AUTOMERGE_TOKEN` can be added if workflow-file
  automation PRs also need to merge automatically under GitHub's stricter
  workflow-permission rules
- the release workflows publish directly to the tagged GitHub Release rather
  than creating a temporary draft shell, and they clean up stale orphan
  `untagged-*` drafts for the same tag before publishing
- release tags are pushed with `RELEASE_AUTOMATION_TOKEN`, the organisation
  secret, because a tag pushed with the repo-scoped `GITHUB_TOKEN` doesn't start
  the workflows that publish it, and a release-bearing commit under
  `.github/workflows/` needs the workflow permission that token carries

Operationally, this means passing dependency updates flow from PR to a
published stable release, APT package and `latest` image without manual
intervention.

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
- the automatic promotion failed partway, so the `Promote Release Candidate`
  workflow is needed to finish it
- a dependency update stays red and should not merge automatically

## What Does Not Create A Release

By default, these changes should not create a new runtime release on their own:

- documentation-only work
- README-only edits
- architecture-only edits
- website-only changes
- Pages-only refreshes
- CI-only changes using non-release-bearing commit types

Those changes may still refresh the landing site from `main`, but they should
not generate a new release tag unless they intentionally change shipped runtime,
container, packaging, or release-pipeline behavior.
