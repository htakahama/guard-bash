#!/bin/bash

# guard-bash 回帰テスト。
# ケースごとに main.sh に JSON を流し、exit code と updatedInput.command を検証する。

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MAIN="$SCRIPT_DIR/main.sh"

# cwd はテスト用に git リポジトリが必要。自身のディレクトリが git 管理下
# であることを前提にする (guard-bash リポジトリ自身)。
TEST_CWD="$SCRIPT_DIR"
if ! git -C "$TEST_CWD" rev-parse --git-dir > /dev/null 2>&1; then
  echo "FATAL: $TEST_CWD is not a git repo. Run 'git init' first." >&2
  exit 1
fi

PASS=0
FAIL=0
FAIL_LOG=""

run_case() {
  local name=$1
  local expect=$2 # allow | block
  local cmd=$3
  local expect_fixed=${4:-}

  local input
  input=$(jq -n --arg cwd "$TEST_CWD" --arg cmd "$cmd" \
    '{cwd:$cwd,tool_input:{command:$cmd}}')

  local stdout stderr rc
  stdout=$(printf '%s' "$input" | "$MAIN" 2> /tmp/guard-bash.test.err)
  rc=$?
  stderr=$(cat /tmp/guard-bash.test.err)

  local verdict
  if [ $rc -eq 0 ]; then
    verdict=allow
  else
    verdict=block
  fi

  local ok=true
  if [ "$verdict" != "$expect" ]; then
    ok=false
  fi

  if [ "$ok" = true ] && [ "$expect" = allow ] && [ -n "$expect_fixed" ]; then
    local got_cmd
    got_cmd=$(printf '%s' "$stdout" | jq -r '.hookSpecificOutput.updatedInput.command')
    if [ "$got_cmd" != "$expect_fixed" ]; then
      ok=false
      stderr="expected cmd: $expect_fixed / got: $got_cmd"
    fi
  fi

  if [ "$ok" = true ]; then
    PASS=$((PASS + 1))
    printf '  ok   %s\n' "$name"
  else
    FAIL=$((FAIL + 1))
    FAIL_LOG+="FAIL: $name"$'\n'"  cmd:      $cmd"$'\n'"  expect:   $expect"$'\n'"  verdict:  $verdict"$'\n'"  stderr:   $stderr"$'\n'"  stdout:   $stdout"$'\n\n'
    printf '  FAIL %s\n' "$name"
  fi
}

echo "Running guard-bash tests (cwd=$TEST_CWD)"

# 正常系
run_case "01 simple git"            allow "git status" \
  "cd $TEST_CWD && git status"
run_case "02 pipe"                  allow "git log | head"
run_case "03 chain"                 allow "git add . && git commit -m foo"
run_case "04 for + cmdsubst"        allow 'for f in $(git ls-files); do cat "$f"; done'
run_case "05 if + test"             allow "if [ -f x ]; then git status; fi"
run_case "06 env FOO=bar"           allow "env FOO=bar git status"
run_case "07 time (TimeClause)"     allow "time git log"
run_case "08 cd under cwd"          allow "cd $TEST_CWD && git status" \
  "cd $TEST_CWD && git status"
run_case "09 nested subshell"       allow "(cd $TEST_CWD && git status)"

# 異常系
run_case "10 cd outside"            block "cd /etc && ls"
run_case "11 for + denied"          block 'for i in 1 2; do sudo reboot; done'
run_case "12 chain + denied"        block "git status && sudo reboot"
run_case "13 dynamic cmd"           block '$cmd arg'
run_case "14 eval denied"           block "eval 'git status'"
run_case "15 parse error"           block "git status '"
run_case "16 unknown cmd"           block "wget2 http://example.com"
run_case "17 cmdsubst denied"       block 'x=$(sudo rm -rf /); echo $x'
run_case "18 denied inside if"      block "if true; then sudo reboot; fi"

echo
echo "Passed: $PASS / Failed: $FAIL"
if [ $FAIL -gt 0 ]; then
  printf '\n%s' "$FAIL_LOG"
  exit 1
fi


# EOF
