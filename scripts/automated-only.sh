#!/usr/bin/env bash
# Say whether a release is made up only of automated updates.
#
# Updates from Dependabot, Renovate and this repository's own workflows release on their own, all
# the way to stable. A release-bearing change that a person merged waits for someone to promote
# it. This checks every release-bearing commit in <since>..<until> and prints automated_only=true
# when each one was authored by one of those accounts, or false when any wasn't, with
# human_commits=<n>. Without <since>, the range is the whole history.
#
# A commit counts as release-bearing if any netspeedy repository's next-release.sh would release
# it: the union of their types (feat, fix, perf, revert, container, build, deps, packaging and
# release), plus breaking changes. So it never treats a commit as non-releasing that one of them
# would release. In a repository that counts fewer types, a person's commit of one of the others
# holds a bot-only release at the candidate, which is the safe side to err on.
#
# It matches the author's email rather than their name. A squash merge takes the name from the PR
# author's profile, which anyone can set to "dependabot[bot]", but the email is the account's
# ID-prefixed noreply address, which nobody else can claim.
#
# Usage: bash scripts/release/automated-only.sh [<since tag>] [<until ref>]
set -euo pipefail

since="${1:-}"
until_ref="${2:-HEAD}"
range="${until_ref}"
if [ -n "${since}" ]; then
  range="${since}..${until_ref}"
fi

human_commits="$(
  git log --format='%ae%x1f%s%x1f%b%x1e' "${range}" | awk '
    BEGIN {
      RS = "\036"; FS = "\037"; count = 0
      automated["49699333+dependabot[bot]@users.noreply.github.com"] = 1
      automated["29139614+renovate[bot]@users.noreply.github.com"] = 1
      automated["41898282+github-actions[bot]@users.noreply.github.com"] = 1
    }
    {
      email = $1
      sub(/^\n+/, "", email)
      subject = $2
      if (email == "") next
      release_bearing = (subject ~ /^(feat|fix|perf|revert|container|build|deps|packaging|release)(\([^)]+\))?!?: / ||
        subject ~ /^[[:alnum:]][[:alnum:]-]*(\([^)]+\))?!: / || $3 ~ /BREAKING CHANGE:/)
      if (!release_bearing) next
      if (!(email in automated)) count++
    }
    END { print count }
  '
)"

if [ "${human_commits}" -eq 0 ]; then
  echo "automated_only=true"
else
  echo "automated_only=false"
fi
echo "human_commits=${human_commits}"
