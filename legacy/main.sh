#!/bin/bash

# guard-bash main.sh
#
# Claude Code の PreToolUse フックとして Bash ツール呼び出しを検証する。
# AST ベース (shfmt -tojson + Python) でコマンドを抽出し、
# denylist / allowlist と突合する。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXTRACT_PY="$SCRIPT_DIR/extract.py"
CHECK_CD_PY="$SCRIPT_DIR/check_cd.py"

# --- allowlist ---
# AST 抽出でコマンドチェーン / for / if / パイプ / コマンド置換の
# 全 CallExpr が検査対象となる。ここに無い名前は即ブロック。
ALLOWED_CMDS=(
  # shell builtin / wrapper
  cd '[' test : true false
  env command nice nohup
  # vcs
  git gh
  # lang / build
  cargo go node npm npx pnpm yarn bun deno
  python python3 pip pip3 uv uvx ruff mypy pyright pytest
  ruby gem bundle rake
  # infra / ops
  docker docker-compose podman kubectl helm terraform
  gcloud firebase gsutil bq
  aws sam cdk
  az func azd
  # shell utilities
  pwd cat head tail less grep rg find ls tree wc diff sort uniq
  sed awk cut tr tee xargs
  echo printf printenv date whoami hostname id which type
  mkdir cp mv ln rm rmdir touch chmod chown
  tar gzip gunzip zip unzip curl wget
  jq yq
  # tools
  mise
  task
  make cmake
  shellcheck shfmt
)

# --- denylist (ALLOWED_CMDS 誤追記時のセーフティネット) ---
DENIED_CMDS=(
  # 破壊的ディスク操作
  dd mkfs fdisk parted mount umount
  # システム制御
  reboot shutdown poweroff halt init systemctl
  # 危険なネットワーク操作
  iptables ip6tables nft tc
  # カーネル / モジュール
  insmod rmmod modprobe sysctl
  # ユーザー / 権限昇格
  sudo su passwd useradd userdel usermod chroot
  # 動的実行 / evaluation
  eval exec source .
)

die() {
  echo "BLOCKED: $*" >&2
  exit 2
}

command -v jq > /dev/null 2>&1 || die "jq is not installed"
command -v shfmt > /dev/null 2>&1 || die "shfmt is not installed"
command -v python3 > /dev/null 2>&1 || die "python3 is not installed"

INPUT=$(cat)
CWD=$(jq -r '.cwd' <<< "$INPUT")
CMD=$(jq -r '.tool_input.command' <<< "$INPUT")
DESC=$(jq -r '.tool_input.description // empty' <<< "$INPUT")

[ -n "$CWD" ] && [ "$CWD" != "null" ] || die "hook input missing .cwd"
[ -n "$CMD" ] && [ "$CMD" != "null" ] || die "hook input missing .tool_input.command"

# 1. cwd が git 管理下か
git -C "$CWD" rev-parse --git-dir > /dev/null 2>&1 \
  || die "cwd '$CWD' is not inside a git repository"

# 2. shfmt で AST を生成 (fail-closed)
if ! AST=$(printf '%s' "$CMD" | shfmt -tojson 2> /tmp/guard-bash.shfmt.err); then
  die "shfmt parse error: $(cat /tmp/guard-bash.shfmt.err 2> /dev/null)"
fi

# 3. コマンド名を抽出
if ! CMDS=$(printf '%s' "$AST" | python3 "$EXTRACT_PY"); then
  die "extract.py failed"
fi

# 4. denylist / allowlist 突合
while IFS= read -r name; do
  [ -n "$name" ] || continue
  if [ "$name" = "__DYNAMIC__" ]; then
    die "dynamic command name (variable / command substitution) is not allowed"
  fi
  for d in "${DENIED_CMDS[@]}"; do
    [ "$name" = "$d" ] && die "'$name' is in the denied command list"
  done
  allowed=false
  for a in "${ALLOWED_CMDS[@]}"; do
    if [ "$name" = "$a" ]; then
      allowed=true
      break
    fi
  done
  if [ "$allowed" = false ]; then
    die "'$name' is not in the allowed command list"
  fi
done <<< "$CMDS"

# 5. cd の先頭判定 (AST ベース)
set +e
CD_STATUS=$(GUARD_CWD="$CWD" python3 "$CHECK_CD_PY" <<< "$AST" 2>&1)
CD_RC=$?
set -e
if [ $CD_RC -ne 0 ]; then
  echo "$CD_STATUS" >&2
  exit 2
fi

case "$CD_STATUS" in
  OK_NO_PREPEND)
    FIXED="$CMD"
    ;;
  OK_PREPEND)
    FIXED="cd $CWD && $CMD"
    ;;
  *)
    die "check_cd.py returned unexpected status: $CD_STATUS"
    ;;
esac

# 6. updatedInput を allow で返す
if [ -n "$DESC" ]; then
  jq -n --arg cmd "$FIXED" --arg desc "$DESC" \
    '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":$cmd,"description":$desc}}}'
else
  jq -n --arg cmd "$FIXED" \
    '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","updatedInput":{"command":$cmd}}}'
fi


# EOF
