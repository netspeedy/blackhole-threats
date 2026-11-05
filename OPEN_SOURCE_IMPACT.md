# Open Source Impact

`blackhole-threats` is MIT-licensed infrastructure software for operators who
want to turn threat intelligence feeds into controlled BGP remote-triggered
blackhole routes.

The project is intentionally public and self-hostable. Operators can inspect
the source, build the binary, run the container image, install the Debian
package, or use the signed APT repository without depending on a proprietary
control plane.

## Public Benefit

The project helps network and security operators:

- deploy a small RTBH route server without building the feed ingestion and BGP
  publication loop from scratch
- consume local, HTTP, and HTTPS threat feeds in plain text, JSON, JSONL, and
  NDJSON formats
- test configuration safely with validation and one-shot run modes
- use first-party Linux binaries, container images, Debian packages, and a
  signed APT repository
- understand release, packaging, and operational behavior through public docs
  and workflows

The likely audience includes homelab operators, small networks, ISPs, managed
service providers, security researchers, and teams that need a transparent
reference implementation for feed-driven RTBH publication.

## Why Support Is Useful

The visible Go daemon is only part of the maintenance cost. Keeping the project
useful as open source also requires:

- CI capacity for formatting, vetting, tests, builds, packaging validation, and
  website checks
- release engineering for release candidates, stable promotion, GitHub Release
  assets, GHCR images, Debian packages, and signed APT metadata
- test environments for Debian package behavior, container startup, and BGP
  smoke tests
- ongoing documentation for operations, configuration, troubleshooting, release
  flow, and deployment examples
- dependency and supply-chain maintenance across Go modules, GitHub Actions,
  container bases, Go build image pins, and website tooling
- security response capacity for private vulnerability reports and dependency
  alerts

Funding, hosted infrastructure, and tooling credits help keep those public
maintenance paths active and reviewable.

## AI And Tooling Credit Use

AI or tooling credits may be used to accelerate public maintenance work such as:

- triaging issues and discussions into clearer reproduction steps
- drafting documentation improvements for maintainer review
- generating test cases and parser fixtures from public or synthetic examples
- reviewing release notes, changelogs, and package metadata for consistency
- summarising CI failures and suggesting focused fixes
- exploring dependency update impact before a maintainer merges changes

AI-assisted output is not treated as an authority. Maintainers remain
responsible for reviewing changes, running validation, preserving release
safety, and deciding what lands in the project.

## Near-Term Outcomes

Additional support would most directly help with:

- expanding feed parser fixtures and route-table reconciliation tests
- improving RouterOS, FRR, bird, and lab deployment examples
- strengthening release artifact and APT repository health checks
- maintaining dependency and supply-chain hardening workflows
- improving troubleshooting docs for first-time operators
- keeping project-health, roadmap, governance, and funding documentation current

These outcomes are also tracked in [ROADMAP.md](ROADMAP.md) and the public
[OSS Roadmap project](https://github.com/users/soakes/projects/1).

## Independence And Boundaries

The project remains maintainer-led. Sponsorship, donations, donated
infrastructure, or credits do not grant private control over:

- the roadmap
- release timing or stable promotion
- signing keys or package publication
- vulnerability handling
- accepted dependencies
- operator-facing behavior

Support should benefit the public repository and the operators who use or learn
from it.

## Current Public Signals

The repository currently provides:

- MIT licensing
- public source, issues, discussions, wiki, and project board
- contribution, support, governance, conduct, and security policies
- private vulnerability reporting
- Dependabot update and security workflows
- release binaries, container images, Debian packages, and signed APT metadata
- public release automation and documentation
