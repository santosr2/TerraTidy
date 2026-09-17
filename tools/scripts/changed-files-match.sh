#!/usr/bin/env bash
# Decide whether the change that triggered a workflow touches files that
# workflow cares about, and write `relevant=true|false` to $GITHUB_OUTPUT.
#
# Workflows used to express this with `paths:` on their triggers, but a workflow
# skipped that way never starts, so its checks never report and a required check
# waits on it forever. Running this as the first job instead lets the workflow
# always start: jobs that don't apply skip themselves, and a final result job
# reports on every change.
#
# Usage: PATTERN='<extended regex>' changed-files-match.sh
#   PATTERN   matched against each changed path (repository-relative)
#
# Reads the standard GitHub Actions environment (GITHUB_EVENT_NAME,
# GITHUB_EVENT_PATH, GITHUB_REPOSITORY, GITHUB_OUTPUT) and needs GH_TOKEN with
# read access to pull requests and contents.
#
# Whenever the changed files can't be determined, it answers `relevant=true`:
# running a workflow that turns out not to be needed is cheap, while skipping
# one that was needed would let an untested change through.
set -euo pipefail

: "${PATTERN:?PATTERN must be set}"

emit() {
  echo "relevant=$1" >> "$GITHUB_OUTPUT"
  echo "relevant=$1"
}

run_everything() {
  echo "::notice::$1; running every job."
  emit true
  exit 0
}

files=""
case "$GITHUB_EVENT_NAME" in
  pull_request | pull_request_target)
    number=$(jq -r '.pull_request.number' "$GITHUB_EVENT_PATH")
    # A renamed file counts under its old path too: moving a file out of a
    # watched directory changes what that workflow covers.
    if ! files=$(gh api --paginate "repos/${GITHUB_REPOSITORY}/pulls/${number}/files?per_page=100" \
        --jq '.[] | .filename, (.previous_filename // empty)'); then
      run_everything "Could not list the files changed by pull request #${number}"
    fi
    ;;
  push)
    before=$(jq -r '.before' "$GITHUB_EVENT_PATH")
    after=$(jq -r '.after' "$GITHUB_EVENT_PATH")
    if [[ "$before" =~ ^0+$ ]]; then
      run_everything "Push created the ref, so there is no previous commit to compare against"
    fi
    # The compare API lists at most 300 files, so a larger push is treated as
    # touching everything rather than as touching only what fit in the list.
    if ! compare=$(gh api "repos/${GITHUB_REPOSITORY}/compare/${before}...${after}" \
        --jq '{count: (.files | length), files: [.files[] | .filename, (.previous_filename // empty)]}'); then
      run_everything "Could not compare ${before}...${after}"
    fi
    if [ "$(jq -r '.count' <<<"$compare")" -ge 300 ]; then
      run_everything "Push changes 300 or more files, more than the compare API lists"
    fi
    files=$(jq -r '.files[]' <<<"$compare")
    ;;
  *)
    run_everything "Event '${GITHUB_EVENT_NAME}' has no change set to filter on"
    ;;
esac

# grep without -q: it reads all of its input, so nothing upstream gets SIGPIPE.
matched=$(grep -E -- "$PATTERN" <<<"$files" || true)

if [ -n "$matched" ]; then
  echo "Relevant changes:"
  while IFS= read -r path; do echo "  $path"; done <<<"$matched"
  emit true
else
  echo "No changed file matches: ${PATTERN}"
  emit false
fi
