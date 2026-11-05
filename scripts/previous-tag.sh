#!/usr/bin/env bash
# Print the release a tag is measured from, for its release notes and its docs archive.
#
# For a release candidate that's the previous candidate of the same version, or the stable
# release before that version when it's the first candidate. For a stable release it's the stable
# release before it. "Before" means lower in version order rather than newest, so a release that's
# published again by hand, after later ones exist, still gets the right base. It prints nothing
# for the first release.
#
# Usage: bash scripts/previous-tag.sh <tag>
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <tag>" >&2
  exit 2
fi
tag="$1"

# The highest of the versions on stdin that sorts below $1, which needn't be among them.
below() {
  { cat; printf '%s\n' "$1"; } \
    | sort -uV \
    | awk -v ref="$1" '$0 == ref { if (previous != "") print previous; exit } { previous = $0 }'
}

stable_tags() {
  git tag --list 'v*' | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true
}

if [[ "${tag}" =~ ^(v[0-9]+\.[0-9]+\.[0-9]+)-rc\.[0-9]+$ ]]; then
  stable="${BASH_REMATCH[1]}"
  previous="$(git tag --list "${stable}-rc.*" | below "${tag}")"
  if [ -z "${previous}" ]; then
    previous="$(stable_tags | below "${stable}")"
  fi
elif [[ "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  previous="$(stable_tags | below "${tag}")"
else
  echo "unsupported release tag: ${tag}" >&2
  exit 1
fi

if [ -n "${previous}" ]; then
  printf '%s\n' "${previous}"
fi
