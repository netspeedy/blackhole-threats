#!/usr/bin/env bash
# Unpack the archived docs of every release line before the latest, for a Pages build.
#
# When a new line ships, Release Assets builds the outgoing line's docs and attaches them to that
# line's last release as blackhole-threats-docs-<line>.tar.gz. This takes each line's newest
# stable release, unpacks its archive, if it has one, into <directory>/<line>/, and prints the
# lines it unpacked as JSON for docs_versions in the site metadata, such as
# [{"version":"v1.1","release":"v1.1.7"}]. The latest tag's own line is the build itself, so it's
# left out.
#
# Deploy Pages Site and Publish Signed Debian Repository both run it, since both deploy the site.
# It needs GH_TOKEN and GITHUB_REPOSITORY.
#
# Usage: bash scripts/download-docs-archives.sh <latest stable tag> <directory>
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "usage: $0 <latest stable tag> <directory>" >&2
  exit 2
fi

latest_tag="$1"
archives="$2"
repo="${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

if [[ ! "${latest_tag}" =~ ^(v[0-9]+\.[0-9]+)\.[0-9]+$ ]]; then
  echo "expected a stable release tag like v1.2.3, got: ${latest_tag}" >&2
  exit 1
fi
current_line="${BASH_REMATCH[1]}"
mkdir -p "${archives}"

# One row per line: the line, its newest stable release and that release's asset names.
lines="$(
  gh api --paginate "repos/${repo}/releases?per_page=100" \
    --jq '.[] | select(.draft == false and .prerelease == false) | [.tag_name, ([.assets[].name] | join(" "))] | @tsv' \
    | sort -t "$(printf '\t')" -k1,1 -rV \
    | awk -F '\t' '{ line = $1; sub(/\.[0-9]+$/, "", line); if (!(line in seen)) { seen[line] = 1; print line "\t" $1 "\t" $2 } }'
)"

versions='[]'
while IFS="$(printf '\t')" read -r line tag assets; do
  asset="${repo#*/}-docs-${line}.tar.gz"
  if [ -z "${line}" ] || [ "${line}" = "${current_line}" ] || [[ " ${assets} " != *" ${asset} "* ]]; then
    continue
  fi
  mkdir -p "${archives}/${line}"
  gh release download "${tag}" --repo "${repo}" --pattern "${asset}" --output - \
    | tar -xz --no-same-owner -C "${archives}/${line}"
  versions="$(jq -c --arg version "${line}" --arg release "${tag}" '. + [{version: $version, release: $release}]' <<<"${versions}")"
done <<<"${lines}"

printf '%s\n' "${versions}"
