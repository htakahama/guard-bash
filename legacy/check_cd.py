#!/usr/bin/env python3
"""
shfmt -tojson の出力を stdin で受け、最上位 Stmt の先頭が cd かどうか、
cd ターゲットが許可 dir 配下かを判定する。

環境変数:
    GUARD_CWD          : Claude Code hook 入力の .cwd (必須)
    GUARD_ALLOWED_DIRS : 追加許可 dir (: 区切り、任意)

Output:
    OK_NO_PREPEND : 先頭が cd <絶対/相対パス> で、パスが許可 dir 配下
    OK_PREPEND    : 先頭が cd ではない (呼び出し側で cd CWD && を付与)
    exit 2        : cd ターゲットが許可外、または動的パス、JSON パース失敗
"""

import json
import os
import sys


def word_to_str(word):
    parts = word.get("Parts", [])
    if not parts:
        return ""
    buf = []
    for p in parts:
        t = p.get("Type")
        if t == "Lit":
            buf.append(p.get("Value", ""))
        elif t == "SglQuoted":
            buf.append(p.get("Value", ""))
        elif t == "DblQuoted":
            for sub in p.get("Parts", []):
                st = sub.get("Type")
                if st == "Lit":
                    buf.append(sub.get("Value", ""))
                else:
                    return None
        else:
            return None
    return "".join(buf)


def unwrap_leftmost(cmd):
    while cmd.get("Type") == "BinaryCmd":
        x = cmd.get("X", {})
        cmd = x.get("Cmd", {})
    return cmd


def extract_cd_target(cmd):
    if cmd.get("Type") != "CallExpr":
        return (False, None)
    args = cmd.get("Args", [])
    if not args:
        return (False, None)
    name = word_to_str(args[0])
    if name != "cd":
        return (False, None)
    if len(args) < 2:
        return (True, None)
    return (True, word_to_str(args[1]))


def is_under(target_abs: str, allowed_abs: str) -> bool:
    try:
        common = os.path.commonpath([target_abs, allowed_abs])
    except ValueError:
        return False
    return common == allowed_abs


def main():
    cwd = os.environ.get("GUARD_CWD", "")
    if not cwd:
        print("check_cd.py: GUARD_CWD is required", file=sys.stderr)
        sys.exit(2)

    allowed = [cwd]
    extra = os.environ.get("GUARD_ALLOWED_DIRS", "")
    if extra:
        allowed += [d for d in extra.split(":") if d]
    allowed_abs = [os.path.realpath(d) for d in allowed]

    try:
        data = json.load(sys.stdin)
    except json.JSONDecodeError as e:
        print(f"check_cd.py: failed to parse JSON: {e}", file=sys.stderr)
        sys.exit(2)

    stmts = data.get("Stmts", [])
    if not stmts:
        print("OK_PREPEND")
        return

    first_cmd = unwrap_leftmost(stmts[0].get("Cmd", {}))
    is_cd, target = extract_cd_target(first_cmd)
    if not is_cd:
        print("OK_PREPEND")
        return

    if target is None:
        print("BLOCKED: cd target is dynamic or missing", file=sys.stderr)
        sys.exit(2)

    if os.path.isabs(target):
        target_abs = os.path.realpath(target)
    else:
        target_abs = os.path.realpath(os.path.join(cwd, target))

    for allowed_dir in allowed_abs:
        if target_abs == allowed_dir or is_under(target_abs, allowed_dir):
            print("OK_NO_PREPEND")
            return

    print(
        f"BLOCKED: cd target '{target}' -> '{target_abs}' is outside allowed dirs: {allowed_abs}",
        file=sys.stderr,
    )
    sys.exit(2)


if __name__ == "__main__":
    main()


# EOF
