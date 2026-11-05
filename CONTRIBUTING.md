# Contributing

Thanks for taking the time to improve `blackhole-threats`. This project is an
operator-facing route server, so changes should be clear, testable, and safe to
ship through the normal release path.

## Ground Rules

- Keep `main` releasable. A successful validation run on `main` can create the
  next release candidate.
- Keep routing behavior explicit and documented. Surprising route announcement
  or withdrawal changes need tests and operator notes.
- Keep packaging, container, Debian, and README behavior aligned when install or
  runtime paths change.
- Treat workflow, release, and signing changes as security-sensitive because
  this is a public repository.
- Report vulnerabilities privately. Do not open a public issue with exploit
  details; see [SECURITY.md](SECURITY.md).

## Before You Start

For small docs fixes, opening a pull request directly is fine. For runtime,
packaging, release, or security-sensitive changes, please open an issue first
or make the pull request description very clear about the operational reason for
the change.

Useful background:

- [README.md](README.md) and the [docs site](https://netspeedy.github.io/blackhole-threats/)
  for operator-facing behavior
- [AGENTS.md](AGENTS.md#project-structure--module-organization) for package
  boundaries
- [docs/release-and-publishing.md](docs/release-and-publishing.md) for release
  automation and publishing rules
- [examples/blackhole-threats.yaml](examples/blackhole-threats.yaml) for the
  reference configuration shape

## Development Setup

Build and validate from the repository root:

```bash
make fmt
make fmt-check
make vet
make test
make build
```

The binary is written to:

```text
dist/blackhole-threats
```

Use the config checker before testing a daemon run:

```bash
./dist/blackhole-threats -conf examples/blackhole-threats.yaml -check-config
```

## Validation Matrix

Run the checks that match the files you changed:

| Change area | Minimum validation |
| --- | --- |
| Go code | `make fmt-check`, `make vet`, `make test`, `make build` |
| Container behavior | `make docker-build` |
| Debian packaging, install paths, service files | `make package` |
| APT repository builder | `bash -n scripts/build-apt-repository.sh` |
| Docs site | `make website-install`, `make website-build`, `make website-check`, `make website-test` |
| Documentation only | Review links and examples for accuracy |

If you cannot run a relevant check locally, say so in the pull request and
explain what you did verify.

## Pull Requests

Pull request descriptions should include:

- the operational reason for the change
- any routing, packaging, upgrade, or security impact
- the exact validation performed
- documentation updates included with the change

Preferred commit and pull request subjects use conventional prefixes such as
`feat:`, `fix:`, `docs:`, `ci:`, `container:`, `packaging:`, or `deps:`.

Release automation treats these prefixes as release-bearing by default:

- `feat:` creates a minor release candidate
- `fix:`, `perf:`, `revert:`, `container:`, `build:`, `deps:`, and
  `packaging:` create patch release candidates
- `BREAKING CHANGE:` in the body or `type!:` in the subject creates a major
  release candidate

These prefixes are normally non-release-bearing:

- `docs:`
- `ci:`
- `chore:`
- `test:`
- `refactor:`

## Documentation

Keep docs close to behavior:

- config schema changes need README, docs, and example config updates
- packaging or service changes need Debian and README updates
- release workflow changes need release documentation updates
- security or support process changes need the root project-health files updated

## Conduct

Participation in this project is covered by
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
