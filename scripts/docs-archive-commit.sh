#!/usr/bin/env bash
# Print the commit an outgoing release line's docs are archived from.
#
# A line's docs stay current on main until the commit that starts the next line: the first
# feat: or breaking change after the line's last release, using the same rules as
# next-release.sh. The archive is built from the commit just before it, so docs-only fixes that
# landed after the last patch release are kept, and the new line's changes are not.
#
# Usage: bash scripts/docs-archive-commit.sh <last tag of the old line> <first tag of the new line>
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <last tag of the old line> <first tag of the new line>" >&2
  exit 2
fi

previous_tag="$1"
next_tag="$2"

while read -r commit; do
  subject="$(git log -1 --format=%s "${commit}")"
  body="$(git log -1 --format=%b "${commit}")"
  if [[ "${subject}" =~ ^feat(\([^\)]+\))?:\  ]] ||
    [[ "${subject}" =~ ^[[:alnum:]][[:alnum:]-]*(\([^\)]+\))?!:\  ]] ||
    grep -q 'BREAKING CHANGE:' <<<"${body}"; then
    git rev-parse "${commit}^"
    exit 0
  fi
done < <(git rev-list --reverse --first-parent "${previous_tag}..${next_tag}")

echo "no feat: or breaking commit between ${previous_tag} and ${next_tag}" >&2
exit 1
