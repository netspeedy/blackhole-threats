# Governance

## Project Model

`blackhole-threats` is a maintainer-led open source project. The code, docs,
packaging, release automation, and website are developed in public under the MIT
License.

The project optimizes for operational reliability over novelty. Changes should
make the daemon safer, clearer, easier to package, or easier to operate.

## Maintainer Responsibilities

Maintainers are responsible for:

- reviewing and merging pull requests
- deciding release timing and stable promotion
- maintaining package, container, and APT repository publication
- keeping public documentation aligned with behavior
- moderating project spaces under the code of conduct
- coordinating private vulnerability reports

## Decision Making

Most decisions are made through issues and pull requests. Maintainers weigh:

- operator safety and clarity
- compatibility with existing deployments
- testability and release risk
- packaging and distribution impact
- long-term maintenance cost

Maintainers may decline changes that increase operational risk, broaden release
or signing permissions unnecessarily, add unclear dependencies, or move the
project away from its route-server purpose.

## Release Authority

The default branch is expected to stay releasable. Once validation succeeds on
`main`, the release automation cuts a release candidate and checks its publish
path. It promotes automated dependency updates to stable on its own; a
maintainer promotes a release that includes changes people merged.

Only maintainers should perform release recovery, signing key management, or
publication changes.

## Funding And Independence

Donations, sponsorships, donated infrastructure, and AI or tooling credits are
welcome when they support the public project. Funding does not buy private
control over the roadmap, maintainer decisions, vulnerability handling, release
channels, or package publication.

Maintainers should disclose material sponsorship or credit arrangements when
they could reasonably affect project priorities.
