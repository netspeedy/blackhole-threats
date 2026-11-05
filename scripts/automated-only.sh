#!/usr/bin/env bash
# Say whether a release is made up only of automated updates.
#
# Updates from Dependabot, Renovate and this repository's own workflows release on their own, all
# the way to stable. A release-bearing change that a person merged waits for someone to promote
# it. This checks every release-bearing commit in <since>..<until>, using the same commit types
# as next-release.sh, and prints automated_only=true when each one was authored by one of those
# accounts, or false when any wasn't, with human_commits=<n>. Without <since>, the range is the
# whole history.
#
# Usage: bash scripts/automated-only.sh [<since tag>] [<until ref>]
set -euo pipefail

since="${1:-}"
until_ref="${2:-HEAD}"
range="${until_ref}"
if [ -n "${since}" ]; then
  range="${since}..${until_ref}"
fi

human_commits="$(
  git log --format='%an%x1f%s%x1f%b%x1e' "${range}" | awk '
    BEGIN { RS = "\036"; FS = "\037"; count = 0 }
    {
      author = $1
      sub(/^\n+/, "", author)
      subject = $2
      if (author == "") next
      release_bearing = (subject ~ /^(feat|fix|perf|revert|container|build|deps|packaging|release)(\([^)]+\))?!?: / ||
        subject ~ /^[[:alnum:]][[:alnum:]-]*(\([^)]+\))?!: / || $3 ~ /BREAKING CHANGE:/)
      if (!release_bearing) next
      if (author != "dependabot[bot]" && author != "renovate[bot]" && author != "github-actions[bot]") count++
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
