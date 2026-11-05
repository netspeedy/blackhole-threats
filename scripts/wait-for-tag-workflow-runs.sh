#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 3 ]; then
  echo "usage: $0 <tag> <head-sha> <workflow> [<workflow> ...]" >&2
  exit 1
fi

tag_name="$1"
head_sha="$2"
shift 2

poll_interval="${POLL_INTERVAL_SECONDS:-3}"
lookup_timeout="${WAIT_FOR_RUN_TIMEOUT_SECONDS:-180}"
lookup_deadline="$(( $(date +%s) + lookup_timeout ))"

find_run_id() {
  local workflow="$1"

  gh run list \
    --workflow "${workflow}" \
    --limit 100 \
    --json databaseId,event,headBranch,headSha \
    | jq -r \
      --arg tag_name "${tag_name}" \
      --arg head_sha "${head_sha}" \
      'map(select(.event == "push" and .headBranch == $tag_name and .headSha == $head_sha)) | first | .databaseId // empty'
}

for workflow in "$@"; do
  run_id=""

  while [ -z "${run_id}" ]; do
    run_id="$(find_run_id "${workflow}")"
    if [ -n "${run_id}" ]; then
      break
    fi

    if [ "$(date +%s)" -ge "${lookup_deadline}" ]; then
      echo "Timed out waiting for push-triggered workflow ${workflow} on tag ${tag_name} (${head_sha})" >&2
      exit 1
    fi

    sleep "${poll_interval}"
  done

  echo "Watching ${workflow} run ${run_id} for ${tag_name}" >&2
  gh run watch "${run_id}" --exit-status
done
