# Support

Support is community-driven and maintainer best-effort. The project is intended
to be understandable enough that operators can validate behavior locally before
running it in production.

## Where To Ask

- Use GitHub Discussions for operator questions, deployment notes, route-policy
  examples, feed ideas, and early roadmap discussion.
- Use GitHub Issues for reproducible bugs, packaging problems, documentation
  gaps, and focused feature requests.
- Use [Troubleshooting](https://netspeedy.github.io/blackhole-threats/guides/troubleshooting/)
  in the docs first for common config, feed, BGP, container, and Debian
  problems.
- Use [SECURITY.md](SECURITY.md) for vulnerabilities or sensitive findings.

## What To Include

For runtime or deployment questions, include:

- `blackhole-threats -version` output or the container/package version
- install method: source, container, Debian package, or APT repository
- operating system and architecture
- redacted YAML configuration, especially BGP ASN, router ID, peers, and feeds
- relevant logs with credentials, private peer details, and sensitive addresses
  removed
- what you expected to happen and what happened instead

For packaging or release issues, include the exact command, package version,
repository URL, architecture, and error output.

## Operational Boundaries

Maintainers can help with project behavior, examples, and packaging bugs. They
cannot safely design or approve a private network's routing policy from a short
public issue or discussion. Treat examples as starting points and validate route
filters in your own environment.

## Funding And Credits

Financial support, hosted credits, and tool credits help cover the ongoing work
around validation, release automation, packaging, documentation, and maintenance.
See [FUNDING.md](FUNDING.md) for the current support model.
