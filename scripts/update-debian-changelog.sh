#!/usr/bin/env bash
set -euo pipefail

release_tag="${1:?stable release tag is required}"
previous_tag="${2:-}"
to_ref="${3:-HEAD}"
suite="${4:-stable}"

if [[ ! "${release_tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "expected a stable release tag like v1.2.3, got: ${release_tag}" >&2
  exit 1
fi

first_line="$(sed -n '1p' debian/changelog)"
package_name="${first_line%% *}"
current_version="$(printf '%s\n' "${first_line}" | sed -n 's/^[^(]*(\([^)]*\)).*/\1/p')"
target_version="${release_tag#v}-1"
urgency="$(printf '%s\n' "${first_line}" | sed -n 's/.*urgency=\([^ ]*\).*/\1/p')"
# sed stops at the first trailer itself. Piping every trailer into head -n1 fails under pipefail
# once there are more than a pipe's buffer of them, because sed is killed by SIGPIPE.
maintainer="$(
  sed -n -e '/^ -- .*  [A-Z][a-z][a-z], /{' -e 's/^ -- \(.*\)  [A-Z][a-z][a-z], .*/\1/p' -e 'q' -e '}' debian/changelog
)"

if [ -z "${package_name}" ] || [ "${package_name}" = "${first_line}" ]; then
  echo "could not parse package name from debian/changelog" >&2
  exit 1
fi
if [ -z "${maintainer}" ]; then
  echo "could not parse maintainer from debian/changelog" >&2
  exit 1
fi

urgency="${urgency:-medium}"

if [ "${current_version}" = "${target_version}" ]; then
  exit 0
fi

log_range="${to_ref}"
if [ -n "${previous_tag}" ]; then
  log_range="${previous_tag}..${to_ref}"
fi

conventional_regex='^([[:alnum:]][[:alnum:]-]*)(\([^)]+\))?(!)?:[[:space:]]+(.*)$'

trim() {
  sed 's/^[[:space:]]*//; s/[[:space:]]*$//'
}

clean_subject() {
  local subject="$1"
  local clean=''

  clean="$(printf '%s' "${subject}" | trim)"

  if [[ "${clean}" =~ ${conventional_regex} ]]; then
    clean="${BASH_REMATCH[4]}"
  fi

  clean="$(printf '%s' "${clean}" | trim)"
  [ -n "${clean}" ] || return 0

  case "${clean}" in
    *[.!?])
      ;;
    *)
      clean="${clean}."
      ;;
  esac

  printf '%s%s\n' \
    "$(printf '%s' "${clean:0:1}" | tr '[:lower:]' '[:upper:]')" \
    "${clean:1}"
}

append_wrapped_bullet() {
  local text="$1"
  local output_path="$2"

  awk -v first_prefix="  * " -v next_prefix="    " -v width=79 '
    {
      line = first_prefix
      count = split($0, words, /[[:space:]]+/)

      for (i = 1; i <= count; i++) {
        if (words[i] == "") {
          continue
        }

        candidate = line
        if (candidate != first_prefix && candidate != next_prefix) {
          candidate = candidate " "
        }
        candidate = candidate words[i]

        if (length(candidate) > width && line != first_prefix && line != next_prefix) {
          print line
          line = next_prefix words[i]
        } else {
          line = candidate
        }
      }

      if (line != first_prefix && line != next_prefix) {
        print line
      }
    }
  ' <<<"${text}" >> "${output_path}"
}

notes_file="$(mktemp)"
tmpfile="$(mktemp)"
trap 'rm -f "${notes_file}" "${tmpfile}"' EXIT

while IFS= read -r -d $'\036' subject; do
  [ -n "${subject}" ] || continue
  subject="${subject#$'\n'}"
  subject="$(printf '%s' "${subject}" | trim)"
  [ -n "${subject}" ] || continue

  if [[ "${subject}" =~ ^Merge\ pull\ request\ #[0-9]+ ]]; then
    continue
  fi

  if [[ "${subject}" =~ ^chore\(release\):[[:space:]]+update\ Debian\ changelog\ for\ v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    continue
  fi

  cleaned="$(clean_subject "${subject}")"
  [ -n "${cleaned}" ] || continue
  append_wrapped_bullet "${cleaned}" "${notes_file}"
done < <(git log --reverse --format='%s%x1e' "${log_range}")

if [ ! -s "${notes_file}" ]; then
  append_wrapped_bullet "Release ${release_tag}." "${notes_file}"
fi

{
  printf '%s (%s) %s; urgency=%s\n\n' \
    "${package_name}" \
    "${target_version}" \
    "${suite}" \
    "${urgency}"
  cat "${notes_file}"
  printf '\n'
  printf ' -- %s  %s\n\n' "${maintainer}" "$(date -R)"
  cat debian/changelog
} > "${tmpfile}"

mv "${tmpfile}" debian/changelog
