# Security Policy

`blackhole-threats` is routing and security-adjacent software. Please report
vulnerabilities responsibly so operators have time to update before details are
public.

## Supported Versions

| Target | Security support |
| --- | --- |
| Latest stable release | Supported with security fixes when practical |
| Current release candidate | Supported as part of the normal prerelease path |
| `main` | Supported for active development and validation |
| Older releases | Not supported unless maintainers explicitly say otherwise |

Security fixes normally flow through `main` and the automated release
candidate path, then a maintainer promotes the candidate to stable.

## Reporting A Vulnerability

Do not open a public issue with exploit details, crash payloads, private
network data, or sensitive logs.

Preferred reporting path:

1. Use GitHub private vulnerability reporting:
   <https://github.com/netspeedy/blackhole-threats/security/advisories/new>.
2. If private vulnerability reporting is unavailable, contact the maintainer
   through a private channel listed on their GitHub profile.
3. If no private channel is available, open a minimal public issue asking for a
   private disclosure route. Do not include technical details in that issue.

Helpful information to include privately:

- affected version, commit, package, or container tag
- install method: source, container, Debian package, or APT repository
- impact and likely exploitability
- reproduction steps or proof of concept
- any relevant logs with secrets, peer details, and private addresses redacted

## Response Expectations

This is a maintainer-led open source project without a guaranteed support SLA,
but security reports are treated as high priority. Maintainers will aim to:

- acknowledge credible reports within a reasonable best-effort window
- confirm affected versions and impact
- prepare a fix and validation path
- coordinate public disclosure once operators have an update path

## Public Security Work

Hardening ideas, dependency updates, documentation improvements, and non-sensitive
security cleanup are welcome as normal public issues or pull requests. Use the
private path only when public details would put operators at risk.
