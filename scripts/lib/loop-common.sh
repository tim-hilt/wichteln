# Shared helpers for scripts/loop.sh and scripts/review-loop.sh; source, don't run.

# Usage: init_loop <directory>
# Checks that <directory> holds issues/ and spec.md, routes every agent commit through
# the pre-commit guardrail (make lint + Rust tests), and sets name (the directory's
# basename), log_file, state_file and resume_session.
init_loop() {
  if [[ -z ${1:-} ]]; then
    echo "Usage: $0 <directory>" >&2
    exit 2
  fi
  local required
  for required in "$1/issues" "$1/spec.md"; do
    if [[ ! -e $required ]]; then
      echo "No such file or directory: $required" >&2
      exit 1
    fi
  done

  git config core.hooksPath .githooks

  name=$(basename "$(cd "$1" && pwd)")
  mkdir -p logs
  log_file="logs/$name.loop.log"
  state_file="logs/$name.loop.state" # the session in flight; left behind if the run is interrupted
  resume_session=$(cat "$state_file" 2>/dev/null) # run_pi continues it instead of starting over
  if [[ -n $resume_session ]]; then echo "Continuing interrupted session $resume_session"; fi
}

# Whether the `**Status:**` line of file $1 is one of the statuses in the regex $2.
has_status() {
  grep -Eiq $'^\\*{0,2}Status:\\*{0,2}[ \t]*('"$2"$')[ \t]*\r?$' "$1"
}

# Appends a titled section to the log file.
log_section() {
  printf '%s\n\n%s\n\n---\n\n' "$1" "$2" >>"$log_file"
}

# --- Status line ---------------------------------------------------------------
# One status at a time: status_set shows it with a spinner and a timer (on a terminal),
# status_stop replaces it with a final line and sets status_elapsed. The timer counts up
# from the start unless a deadline is given, then it counts down to it.

status_message=""
status_deadline=""
spinner_pid=""

format_elapsed() {
  local minutes=$(($1 / 60)) seconds=$(($1 % 60))
  if ((minutes > 0)); then echo "${minutes}m ${seconds}s"; else echo "${seconds}s"; fi
}

# Formats the seconds left until a deadline as "1h 9m left", "9m 5s left" or "5s left".
format_remaining() {
  local seconds=$1 hours minutes
  ((seconds > 0)) || seconds=0
  hours=$((seconds / 3600)) minutes=$((seconds % 3600 / 60))
  if ((hours > 0)); then
    echo "${hours}h ${minutes}m left"
  elif ((minutes > 0)); then
    echo "${minutes}m $((seconds % 60))s left"
  else
    echo "${seconds}s left"
  fi
}

# The spinner's timer: time left until status_deadline, else time since status_start.
status_timer() {
  if [[ -n $status_deadline ]]; then
    format_remaining $((status_deadline - $(date +%s)))
  else
    format_elapsed $(($(date +%s) - status_start))
  fi
}

# Usage: status_set <message> [deadline-epoch]
# Shows <message> behind a spinner, counting down to <deadline-epoch> when it is given
# and counting elapsed time otherwise.
status_set() {
  status_message=$1
  status_deadline=${2:-}
  status_start=$(date +%s)
  [[ -t 1 ]] || return 0
  (
    frames=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    frame=0
    while :; do
      printf '\r\033[K%s %s (%s)' "${frames[frame++ % 10]}" "$status_message" \
        "$(status_timer)"
      sleep 0.1
    done
  ) &
  spinner_pid=$!
}

# No-op when no status is shown.
status_stop() {
  [[ -n $status_message ]] || return 0
  if [[ -n $spinner_pid ]]; then
    kill "$spinner_pid" 2>/dev/null
    wait "$spinner_pid" 2>/dev/null
    spinner_pid=""
    printf '\r\033[K'
  fi
  status_elapsed=$(format_elapsed $(($(date +%s) - status_start)))
  echo "$status_message ($status_elapsed)"
  status_message=""
}

trap '[[ -n $spinner_pid ]] && kill "$spinner_pid" 2>/dev/null' EXIT

# Usage: gate_on_tests <what-just-ran>
# The pre-commit hook skips the Apple tests, so the loops run them after every session.
# On a failure an agent gets the output, fixes and commits, and the tests run again,
# until they pass or $max_fix_attempts fixes have not helped (then the loop stops).
max_fix_attempts=${MAX_FIX_ATTEMPTS:-10}
gate_on_tests() {
  local output attempt=0 failing
  while :; do
    status_set "Testing after $1"
    output=$(make test 2>&1) && { status_stop; return 0; }
    status_stop
    log_section "make test failed after $1 (fix attempts so far: $attempt)" "$output"
    if ((attempt >= max_fix_attempts)); then
      printf 'make test still failing after %s fix attempts for %s; stopping (see %s)\n\n%s\n' \
        "$attempt" "$1" "$log_file" "$(tail -n 40 <<<"$output")" >&2
      exit 1
    fi
    attempt=$((attempt + 1))
    # Drop the known display-link noise, keep the tail where the failure is reported.
    failing=$(grep -v 'CVDisplayLinkCreateWithCGDisplays' <<<"$output" | tail -n 150)
    run_pi "Fixing tests after $1 (attempt $attempt)" claude-sonnet-5-5 medium \
      "fix-$name-$(basename "$1" .md)-$attempt" \
      "/skill:diagnosing-bugs \`make test\` failed after $1. Find the root cause and fix it; do not weaken or delete tests to make them pass. Full output is in /tmp/pkm-test.log; the tail follows:

$failing

Run \`make test\` yourself to check the fix, then commit using the commit-message 'Fix tests after $1'."
    log_section "Fix attempt $attempt after $1 took $status_elapsed" "$pi_output"
  done
}

# --- Claude subscription quota -------------------------------------------------
# A Claude subscription answers 429 once its rolling 5-hour ("session") or weekly window
# is spent, and pi's error carries no reset time. The OAuth usage endpoint reports each
# window's utilization and resets_at.

# Waits out an exhausted 5-hour window, and stops the loop when a weekly window is
# exhausted (waiting days is not the loop's job). Returns 1 without waiting when the
# subscription has headroom or its usage can't be read.
await_quota() {
  local token usage window reset
  token=$(pi auth print-bearer-token --provider anthropic 2>/dev/null) && [[ -n $token ]] || return 1
  usage=$(curl -fsS --max-time 20 https://api.anthropic.com/api/oauth/usage \
    -H "Authorization: Bearer $token" \
    -H "anthropic-beta: oauth-2025-04-20" \
    -H "User-Agent: claude-code/2.0.32") || return 1

  # The exhausted window that lifts last, weekly first: "<group> <reset-epoch>".
  read -r window reset < <(jq -r '
    [.limits[]? | select(.percent >= 99 and .resets_at != null)
      | {group, reset: (.resets_at | sub("\\.[0-9]+"; "") | sub("\\+00:00$"; "Z") | fromdate)}]
    | ((map(select(.group == "weekly")) | max_by(.reset)) // max_by(.reset) // empty)
    | "\(.group) \(.reset)"' <<<"$usage" 2>/dev/null)
  [[ -n $reset ]] || return 1

  local until
  until=$(date -r "$reset" '+%Y-%m-%d %H:%M')
  if [[ $window == weekly ]]; then
    log_section "Claude weekly window exhausted until $until; stopping" "$usage"
    printf 'Claude weekly limit exhausted until %s; re-run the loop after that (usage in %s)\n' "$until" "$log_file" >&2
    exit 1
  fi

  log_section "Claude 5-hour window exhausted; waiting until $until" "$usage"
  # At least a minute, so a resets_at in the past is not probed in a hot loop. Short
  # sleeps against the wall clock, since `sleep` stalls while the laptop is suspended.
  local wake=$((reset + 90)) floor=$(($(date +%s) + 60))
  ((wake > floor)) || wake=$floor
  status_set "Claude 5-hour window exhausted; waiting until $until" "$wake"
  while (($(date +%s) < wake)); do sleep 60; done
  status_stop
}

# Usage: run_pi <status> <model> <thinking> <session-id> <prompt>
# Runs pi headlessly in that session behind the status line <status>, then sets pi_output
# to pi's combined output and status_elapsed to the run's duration. A session an
# interrupted earlier run left behind is continued instead of started over. Waits out
# Claude rate limits and continues the session afterwards; exits on any other failure.
run_pi() {
  local status=$1 model=$2 thinking=$3 session=$4 prompt=$5 code
  if [[ $session == "$resume_session" ]]; then
    prompt="continue"
    resume_session=""
  fi

  await_quota # don't spend a session on an immediate 429
  while :; do
    status_set "$status"
    printf '%s\n' "$session" >"$state_file"
    pi_output=$(pi -p "$prompt" --session-id "$session" \
      --provider anthropic --model "$model" --thinking "$thinking" </dev/null 2>&1)
    code=$?
    status_stop
    if ((code == 0)); then
      rm -f "$state_file"
      return 0
    fi
    if ! grep -qE '^429 |rate_limit_error' <<<"$pi_output"; then
      printf 'pi exited with status %s\n\n%s\n' "$code" "$pi_output" >&2
      exit 1
    fi

    log_section "pi hit a Claude rate limit (status $code)" "$pi_output"
    if ! await_quota; then # throttled with headroom left, or usage unknown
      status_set "Claude throttled; retrying in 5m"
      sleep 300
      status_stop
    fi
    prompt="continue"
  done
}
