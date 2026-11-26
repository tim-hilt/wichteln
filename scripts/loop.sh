#!/usr/bin/env bash
# Usage: scripts/loop.sh <project-directory>
#
# Implements the ready-for-agent tasks in <project-directory>/issues/ one pi session
# each, in file order and after their blockers, then reviews them all against
# <project-directory>/spec.md after every $review_every implemented tasks (unless only one task is left, which
# joins that review) and once more before exit if tasks remain unreviewed.

set -uo pipefail

source "$(dirname "$0")/lib/loop-common.sh"

directory=${1:-} # kept as given, so paths stay relative to the cwd
init_loop "$directory"
tasks_dir="$directory/issues"
spec_file="$directory/spec.md"
review_session="review-$name"
review_every=5

shopt -s nullglob
tasks=("$tasks_dir"/*.md) # glob expansion is sorted
shopt -u nullglob

# The ticket numbers on the `**Blocked by:**` line of $1 that are not yet
# implemented (Status in-review or done), space-separated; empty if none.
pending_blockers() {
  local number blocker pending=""
  for number in $(grep -Ei '^\*{0,2}Blocked by:' "$1" | grep -oE '\b[0-9]{2}\b'); do
    blocker=$(compgen -G "$tasks_dir/$number-*.md" | head -n 1)
    if [[ -z $blocker ]] || ! has_status "$blocker" 'in-review|done'; then
      pending+="$number "
    fi
  done
  echo "${pending% }"
}

# Writes to $1 the log sections since the last review: implementations,
# without quota noise, so a review doesn't read the whole log.
write_review_context() {
  awk '
    /^[^ ]+ reviewed in [0-9]/ { kept = ""; keep = 0; next }
    /^[^ ]+\.md implemented in / { keep = 1 }
    /^(Claude (weekly|5-hour) window exhausted|pi hit a Claude rate limit|Still blocked, not implemented)/ { keep = 0 }
    keep { kept = kept $0 "\n" }
    END { printf "%s", kept }
  ' "$log_file" >"$1"
}

# Reviews the tasks implemented since the last review. An interrupted review
# is resumed first, under its own session id.
reviews=0
unreviewed=0
review() {
  local session="$review_session-$((reviews + 1))"
  if [[ $resume_session == "$review_session"* ]]; then session=$resume_session; fi
  local context_file="logs/$name/$session.context.log"
  mkdir -p "logs/$name"
  write_review_context "$context_file"
  run_pi "Reviewing $spec_file" claude-opus-5-5 high "$session" \
    "/skill:software-design the last $unreviewed commits implemented the tickets in @$tasks_dir. Please review \`git diff HEAD~$unreviewed\` according to @$spec_file. Also consider @$context_file, the loop log since the last review, to understand how we got to the current state. Pay special attention to correctness and simplicity. Fix found issues right away. If an issue is fully implemented, then please set its Status line to 'done'. Commit your changes using the commit-message 'Reviewed $spec_file'."
  log_section "$spec_file reviewed in $status_elapsed" "$pi_output"
  reviews=$((reviews + 1))
  unreviewed=0
}

# Number of ready-for-agent tasks not yet run.
remaining_tasks() {
  local count=0 task_file
  for task_file in ${tasks[@]+"${tasks[@]}"}; do
    has_status "$task_file" 'ready-for-agent' || continue
    [[ $attempted == *" $task_file "* ]] || count=$((count + 1))
  done
  echo "$count"
}

implemented=0
attempted=" " # tasks already run: one whose Status the agent did not update runs only once

# Passes repeat until one implements nothing, so a task blocked by a
# later-numbered task runs in the pass after its blocker.
progress=1
while ((progress)); do
  progress=0
  for task_file in ${tasks[@]+"${tasks[@]}"}; do
    # Re-read the Status every time; earlier implementations may change later tasks.
    has_status "$task_file" 'ready-for-agent' || continue
    [[ $attempted == *" $task_file "* ]] && continue
    [[ -z $(pending_blockers "$task_file") ]] || continue
    attempted+="$task_file "

    run_pi "Implementing $task_file" claude-sonnet-5-5 medium "loop-$name-$(basename "$task_file" .md)" \
      "/skill:tdd implement the task described in @$task_file. Apply software-design best practices during the implementation. Tick off all completed bullet points in $task_file and set its Status line to in-review when you're finished; edit it in place, since .scratch/ is untracked. Commit the implementation using the commit-message 'Implement $task_file'."
    log_section "$task_file implemented in $status_elapsed" "$pi_output"

    implemented=$((implemented + 1))
    unreviewed=$((unreviewed + 1))
    progress=1
    # A single task left joins this batch's review instead of getting its own.
    if ((unreviewed >= review_every && $(remaining_tasks) != 1)); then review; fi
  done
done

still_blocked=""
for task_file in ${tasks[@]+"${tasks[@]}"}; do
  has_status "$task_file" 'ready-for-agent' || continue
  [[ $attempted == *" $task_file "* ]] && continue
  still_blocked+="$task_file (waits for $(pending_blockers "$task_file"))"$'\n'
done
if [[ -n $still_blocked ]]; then
  log_section "Still blocked, not implemented" "$still_blocked"
  printf 'Still blocked, not implemented:\n%s' "$still_blocked" >&2
fi

# An interrupted review is finished even when there is nothing new to review.
if ((unreviewed == 0)) && [[ $resume_session != "$review_session"* ]]; then
  ((implemented == 0)) && echo "No ready-for-agent tasks found."
  exit 0
fi

review

if [[ -n $resume_session ]]; then
  printf 'Interrupted session %s was not resumed; its task is no longer ready-for-agent. Open it with: pi --session-id %s\n' \
    "$resume_session" "$resume_session" >&2
fi
