# Repository Guidelines

## Project Structure & Module Organization

This repository is a single Go module with first-party packaging and release
automation.

- Keep the executable entrypoint in `cmd/blackhole-threats/`.
- Keep application code under `internal/`; do not add new top-level Go packages
  at the repo root.
- Keep root-level files for module, build, packaging, release metadata, and
  open-source project health only: `go.mod`, `go.sum`, `Makefile`,
  `Dockerfile`, `README.md`, `LICENSE`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `SECURITY.md`, `SUPPORT.md`, `GOVERNANCE.md`,
  `FUNDING.md`, `ROADMAP.md`, `OPEN_SOURCE_IMPACT.md`, and repo config files
  belong at the root because local tooling, GitHub community-profile checks, and
  funding reviewers expect them there.
- `internal/buildinfo` is the canonical location for version, commit, and build
  date metadata injected at build time.
- `internal/config` owns YAML loading, feed definitions, and BGP community
  parsing.
- `internal/feed` owns feed retrieval, text/JSON parsing, and prefix
  summarisation.
- `internal/bgp` owns GoBGP lifecycle and route announce/withdraw logic.
- Keep operator-facing sample configuration in `examples/`.
- Keep container runtime assets under `packaging/container/`; the Docker image
  root filesystem and S6 service definitions belong there, not inline in the
  workflow files.
- Keep Debian packaging metadata in `debian/`.
- Keep release and repository automation scripts in `scripts/`.
- Keep architecture and design notes in `docs/`.
- Keep the GitHub Pages landing site in `website/`; treat it as
  the public frontend for the signed APT repository rather than as a throwaway
  marketing stub. Keep it out of `.github/`: Dependabot stopped being able to
  open npm pull requests for manifests under `.github/` in June 2026 (its
  create-PR call is rejected as "invalid or unauthorized changes").
- Retired files should be removed from the working tree rather than left behind
  as dead alternatives.

## Build, Test, and Development Commands

- Format Go code:
  ```bash
  make fmt
  ```
- Check formatting without rewriting files:
  ```bash
  make fmt-check
  ```
- Run static validation:
  ```bash
  make vet
  ```
- Run tests:
  ```bash
  make test
  ```
- Build the local binary:
  ```bash
  make build
  ```
- Build a cross-compiled binary:
  ```bash
  make build-cross GOOS=linux GOARCH=arm64 OUTPUT=dist/blackhole-threats-linux-arm64
  ```
- Build the container image locally:
  ```bash
  make docker-build
  ```
- Build the Debian package locally:
  ```bash
  make package
  ```
- Build the GitHub Pages site locally:
  ```bash
  make website-install
  make website-build
  ```
- When editing the APT repository builder, at minimum syntax-check it:
  ```bash
  bash -n scripts/build-apt-repository.sh
  ```

## Coding Style & Naming Conventions

- Follow `.editorconfig`: Markdown, YAML, and JSON use 2-space indentation;
  Makefiles use tabs.
- Always run `gofmt`; do not hand-format Go code.
- Prefer small, focused internal packages with clear ownership rather than large
  shared utility layers.
- Keep public surface area small; prefer `internal/...` over adding new exported
  packages unless there is a clear reason.
- Prefer explicit, wrapped errors that preserve the failing path or operation.
- Keep logging operator-focused and actionable.
- Keep configuration examples realistic and ready to copy, not placeholder-heavy.
- Prefer descriptive workflow filenames and workflow `name:` values such as
  `build-and-validate.yml` or `publish-apt-repository.yml`; avoid vague names
  like `ci.yml` when the workflow has a narrower responsibility.
- Keep file and directory names stable when they are referenced by packaging,
  workflows, or documentation.

## Versioning, Release, and Packaging Rules

- The default branch is `main`.
- Stable release tags are `v<major>.<minor>.<patch>` and prerelease tags are
  `v<major>.<minor>.<patch>-rc.<n>`; do not invent a parallel tag scheme
  without updating the workflows, README, and release docs together.
- Local and CI builds derive version metadata from git and inject it through
  linker flags into `internal/buildinfo`; do not hardcode release versions in Go
  source files.
- `go.mod` tracks the minimum supported Go version for source builds and Debian
  packaging. Keep it buildable with the Go toolchain shipped by Debian trixie
  unless the Debian packaging strategy changes deliberately.
- GitHub Actions source builds may track the current stable Go release
  separately from `go.mod`, and the Docker build stage keeps its own pinned
  `GO_VERSION` in `Dockerfile`. Do not use the `go` directive in `go.mod` as a
  proxy for "latest CI Go version".
- Current release automation publishes Linux binaries for `amd64`, `arm64`, and
  `arm`, Debian binary packages for `amd64`, `arm64`, and `armhf`, Debian
  source package artifacts for tagged releases, and signed APT metadata
  including source indexes.
- Keep GitHub Release assets curated for operators: upload the runtime tarballs,
  non-dbgsym Debian packages, `sha256sums.txt`, and the Debian source package
  trio, but do not attach maintainer-oriented byproducts such as `-dbgsym`,
  `.buildinfo`, `.changes`, or a duplicate `release-notes.md` asset.
- `main` auto-cuts release candidates after `Build and Validate` succeeds and
  validates the prerelease asset and container publish paths, then promotes the
  candidate to stable on the same commit and waits for the stable publish
  workflows (release assets, container image, APT repository). Nobody presses a
  button: a merged release-bearing change ships on its own, so keep `main`
  releasable. The `Promote Release Candidate` workflow is only for recovery.
- The checked-in `debian/changelog` must track the latest stable package
  version present on `main`. Automated release runs update it with
  `scripts/update-debian-changelog.sh` on a `chore(release): update Debian
  changelog for vX.Y.Z` commit before tagging, then wait for the changelog
  commit to pass `Build and Validate`.
- Because automated release candidates and their matching stable releases point
  at the same commit, the source changelog records the stable Debian version
  such as `1.2.3-1`; release artifact jobs may still rewrite the build-local
  changelog to `1.2.3~rc.N-1` through `scripts/prepare-debian-changelog.sh`
  when building RC packages.
- Release-note formatting is driven by `.github/release-drafter.yml`, the
  repository label catalog in `.github/repository-labels.json`, and the
  `Release Drafter` workflow. Keep those aligned when labels, categories, or
  version-bump expectations change.
- Keep `.github/release-drafter.yml` aligned with `scripts/next-release.sh`.
  Labels such as `documentation`, `ci`, and `maintenance` may still appear in
  grouped release notes, but they must not request a version bump on their own.
- `Makefile` is the source of truth for local build targets and build metadata
  wiring. Keep new local build or validation steps there when they are intended
  for contributors.
- Routine local builds should use Go's default cache locations. Do not add a
  repo-root `.cache/` workflow back to the top-level `Makefile` unless there is
  a specific CI or packaging need.
- Keep the binary name `blackhole-threats` unless the repository, Debian
  package, container image, service name, and documentation are all updated in
  the same change.
- Debian package installs should remain conventional and documented. If install
  paths change, update the packaging metadata and the README Debian section in
  the same change.
- Current Debian paths are:
  - `/usr/sbin/blackhole-threats`
  - `/etc/blackhole-threats.yaml`
  - `/etc/default/blackhole-threats`
  - `/usr/share/man/man8/blackhole-threats.8.gz`
  - `/usr/share/doc/blackhole-threats/examples/blackhole-threats.yaml`
- The Debian package and the GitHub Pages APT repository are both first-party
  deliverables. Do not treat Debian packaging as secondary or optional.
- The GitHub Pages root is both a human-facing landing site and the machine-
  readable APT repository root. Keep the landing page compatible with static
  hosting and do not break the `dists/`, `pool/`, or key download paths.
- Website-only GitHub Pages refreshes from `main` should reuse the latest
  published repository snapshot rather than forcing a new release tag or
  rebuilding unsigned repository metadata.
- Documentation-only, README-only, architecture-only, website-only, and
  Pages-only changes are non-release-bearing: they never start a release on
  their own, and they refresh the GitHub Pages site from `main`.
- `scripts/build-apt-repository.sh` is expected to keep publishing both binary
  and source package indexes. Do not regress `deb-src` support.
- Keep Debian changelog entries factual, operator-facing, and suitable for an
  actual package archive; avoid speculative or placeholder wording.
- Keep `debian/control`, `debian/rules`, service files, defaults, and the README
  aligned whenever packaging behavior changes.

## Container Rules

- GHCR is the canonical published registry for this repository's container image.
- Keep the published image reference aligned with `ghcr.io/netspeedy/blackhole-threats`
  unless the repository owner or publishing design changes intentionally.
- Keep the Docker builder and runtime bases aligned with the supported Debian
  release line used by the repo.
- Keep the pinned `S6_OVERLAY_VERSION` and the matching checksum pins in
  `Dockerfile`; the scheduled refresh workflow updates them together.
- Container bootstrap behavior should continue to create or expose the sample
  config through `/config` as documented and smoke-tested by CI.
- Keep container platform coverage at `linux/amd64` and `linux/arm64` unless the
  workflows and README are updated together.
- If container paths, labels, runtime user expectations, or startup behavior
  change, update `Dockerfile`, `packaging/container/`, the container workflow,
  and the README container section together.

## GitHub Actions & Public Repo Security

- Treat this as a public repository. Favor least privilege and assume workflow
  changes are security-sensitive.
- Pull requests should validate code and packaging, but should not gain access
  to signing or publish credentials.
- Do not broaden workflow `permissions:` unless there is a clear operational
  need and the change is documented.
- Keep write permissions scoped to the specific publish jobs that need them.
- Container publishing is limited to trusted `v*` / `v*-rc.*` tags and
  explicit recovery dispatches; pull requests run validation-only container
  builds.
- Release publishing is limited to trusted `v*` / `v*-rc.*` tags.
- Signed APT repository publishing is limited to trusted stable `v*` tags.
- The separate landing-site Pages deploy may run from `main` pushes and after
  successful stable APT publication, but it must only republish the website on
  top of the latest published repository snapshot; do not turn those refreshes
  into unsigned APT repository rebuilds.
- Release and publish tags must point to commits already contained in `main`;
  keep that verification in place.
- Keep the scheduled toolchain/runtime refresh workflow branch-scoped and
  PR-based; it should update the root pins and open a reviewable pull request
  rather than pushing directly to `main`.
- Dependabot auto-merge should only act after `Build and Validate` succeeds for
  the PR head revision, and it should preserve the Dependabot pull request
  title as the squash commit subject so existing `deps:` / `container:` release
  automation still works after merge.
- Dependabot version-update automation intentionally does not semver-filter PRs:
  patch, minor, and major updates are all eligible to merge after successful
  validation, and the repo keeps enough Dependabot PR capacity that a failing
  major update does not block fresh update proposals.
- Keep the APT signing material in the protected `apt-repository` GitHub Actions
  environment. Do not move archive-signing secrets to broad repository-level
  secrets.
- Prefer the repo-scoped `GITHUB_TOKEN` for in-repo Dependabot merge automation
  under the current repository settings. Do not introduce a long-lived personal
  access token just to merge Dependabot PRs unless branch protection or GitHub
  App policy changes make that necessary. If workflow-file Dependabot PRs also
  need to merge automatically, the supported exception is an optional
  `AUTOMATION_AUTOMERGE_TOKEN` with the minimum extra workflow permission needed
  for that class of PR.
- Automated releases may need `RELEASE_AUTOMATION_TOKEN` to push the generated
  `debian/changelog` commit and then tag the validated follow-up commit. Keep
  that token scoped to the minimum repository contents and workflow permission
  needed for release automation; do not introduce broader personal tokens.
- If you change workflow behavior, update the README CI/CD, APT repository, or
  contribution sections in the same change when operator behavior changes.

## Testing Guidelines

- Minimum validation before opening or updating a PR that changes Go code:
  ```bash
  make fmt-check
  make vet
  make test
  make build
  ```
- If you change container behavior, also run:
  ```bash
  make docker-build
  ```
- If you change Debian packaging, install paths, service files, or release
  packaging, also run:
  ```bash
  make package
  ```
- If you change `website/` or GitHub Pages publishing behavior,
  also run:
  ```bash
  make website-install
  make website-check
  make website-build
  ```
- If you change `scripts/build-apt-repository.sh`, also run:
  ```bash
  bash -n scripts/build-apt-repository.sh
  ```
- Prefer targeted validation that matches the files changed, but do not skip the
  relevant packaging or publish-path checks when release behavior is affected.

## Commit & Pull Request Guidelines

- Do not create commits unless the user explicitly asks for a commit in the
  current turn after reviewing the changes.
- Keep changes focused by concern: runtime logic, packaging, container, docs,
  and workflow hardening should be split when practical.
- Split commits by concern when that makes review or future history clearer.
  Do not combine unrelated cleanup, docs, workflow, and runtime changes into a
  single commit when they can be separated cleanly.
- Commits intended for `main` should follow a conventional format such as
  `feat: ...`, `fix: ...`, `docs: ...`, `ci: ...`, or `container: ...`.
- Write a meaningful commit body whenever the change affects operators,
  packaging, release behavior, or security posture. Automated versioning still
  reads commit history, and the fallback release-note path uses commit bodies
  when a change lands on `main` without a pull request.
- Automated version bumps treat `feat` as minor, `fix`/`perf`/`revert`/
  `container`/`build`/`deps`/`packaging` as patch, and `BREAKING CHANGE:` or
  `type!:` as major. `docs`, `ci`, `chore`, `test`, and `refactor` do not cut a
  release by default unless they are intentionally expressed with a
  release-bearing type.
- Pull request labels drive the polished GitHub release notes. Prefer the
  correct release-note label even when the autolabeler would infer it, and fix
  labels before merge rather than relying on the direct-commit fallback.
- Pull request titles and bodies should read like standard maintainer-authored
  communication. Do not mention `AGENTS.md`, prompts, AI assistance, agent
  workflow, or other internal instruction sources in PR content; state the
  rationale, release impact, and validation directly.
- If you choose a subject prefix, prefer clear conventional-style prefixes.
  Existing repo automation already uses `chore`, `ci`, `deps`, and `container`.
- If distinct changes should remain separate in `main`, prefer separate PRs or
  another merge approach that preserves commit boundaries rather than relying on
  a squash merge to keep them separate.
- Treat `main` as release-candidate-ready: after `Build and Validate` passes
  for a push to `main`, the release workflow can automatically create the next
  `v*-rc.*` tag from merged commit history and run the prerelease publish path.
  Do not merge partial work to `main` that should not become a published
  prerelease.
- Docs-only, README-only, website-only, or CI-only landing-page updates should
  use the normal `main` push path and the Pages-site deploy workflow; do not
  cut a release tag just to refresh the public website.
- Do not hand-push routine `v*` or `v*-rc.*` tags. Automated tagging from
  `main` is the normal path, and it promotes to stable itself; the manual
  promotion workflow and manual tags are for recovering from a failed run.
- Pull requests should explain:
  - the operational reason for the change
  - any routing, packaging, or upgrade impact
  - the exact validation performed
  - any documentation updates included
- Do not bundle unrelated workflow, packaging, and runtime refactors into one
  PR without a good reason.

## Documentation Sync

- Keep `README.md` aligned with the actual operator workflow. If install,
  release, container, or repository behavior changes, update the relevant README
  sections in the same change.
- Keep `docs/architecture.md` aligned with the real package/module boundaries
  when code ownership moves.
- If the configuration schema or operator defaults change, update
  `examples/blackhole-threats.yaml` and the matching README configuration text
  in the same change.
- If Debian-installed paths or service behavior change, update the README Debian
  package section.
- If APT publication, signing, or Pages behavior changes, update the README APT
  repository section.
- If the `make website-*` targets, the site's build inputs, or the workflows that
  publish it change, update `website/README.md`.
- If workflow names, publish conditions, or security posture change, update the
  README CI/CD and contributing sections.
- If contribution, conduct, security, support, governance, roadmap, funding, or
  impact expectations change, update the matching root project-health file and
  the README open-source support section in the same change.

## Agent Behaviour & Task Scope

These rules apply specifically to AI coding agents working in this repository.

- Always read this file first before starting a task.
- After creating or updating a PR branch and pushing it, switch the local
  checkout back to `main` unless the user explicitly asks to stay on the
  feature branch.
- Prefer targeted changes over wide refactors.
- Do not move `go.mod`, `go.sum`, `Makefile`, `Dockerfile`, or the `debian/`
  directory without an explicit task that requires it.
- Do not weaken release, signing, or public-repo safety checks just to make CI
  simpler.
- Use subagents when it materially speeds up parallel-safe exploration or
  validation, but keep the immediate blocking implementation path in the main
  agent.
- When changing release or packaging behavior, inspect the matching workflow
  files rather than assuming the README is sufficient.
- When changing runtime behavior, inspect the relevant code under `cmd/` and
  `internal/` before editing docs.
- Do not introduce new third-party services, registries, or release channels
  without reflecting that in workflows, docs, and packaging.
- Leave the worktree ready for user review; do not create a commit unless asked.

## Exploration Checklist

Before making non-trivial changes, an agent should:

1. Read `README.md` and `docs/architecture.md` for the current operator-facing
   conventions.
2. Review `Makefile` before inventing new local build or test commands.
3. Review the relevant files under `.github/workflows/` before changing release,
   packaging, or publish behavior.
4. Review `debian/` before changing install paths, service behavior, or Debian
   metadata.
5. Review `packaging/container/` and `Dockerfile` together before changing
   container startup or filesystem layout.

## Self-Maintenance

- Keep this file current.
- If a repo convention changes, update this file in the same change.
- If a recurring misunderstanding is corrected during a task, document the
  corrected rule here.
- Do not remove an outdated rule without replacing it with the new repo truth.
