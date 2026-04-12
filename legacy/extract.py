#!/usr/bin/env python3
"""
shfmt -tojson の出力を stdin で受け、AST 中の全 CallExpr からコマンド名を
行区切りで stdout に出力する。

動的語 (ParamExp / CmdSubst / ArithmExp / ProcSubst を含む) は
"__DYNAMIC__" を出力する。JSON パース失敗時は exit 2。
"""

import json
import os
import sys

DYNAMIC = "__DYNAMIC__"

WRAPPERS = {"env", "command", "nice", "nohup"}


def is_assign_token(s: str) -> bool:
    if "=" not in s:
        return False
    name = s.split("=", 1)[0]
    return bool(name) and name.replace("_", "a").isalnum() and not name[0].isdigit()


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


def basename(path: str) -> str:
    return os.path.basename(path) if "/" in path else path


def handle_callexpr(node, out):
    args = node.get("Args", [])
    if not args:
        return
    first = word_to_str(args[0])
    if first is None:
        out.append(DYNAMIC)
        return
    if not first:
        return
    first = basename(first)
    out.append(first)

    if first in WRAPPERS:
        for idx in range(1, len(args)):
            w = word_to_str(args[idx])
            if w is None:
                out.append(DYNAMIC)
                return
            if first == "env" and is_assign_token(w):
                continue
            if w.startswith("-"):
                continue
            out.append(basename(w))
            return


def walk(node, out):
    if isinstance(node, dict):
        if node.get("Type") == "CallExpr":
            handle_callexpr(node, out)
        for v in node.values():
            walk(v, out)
    elif isinstance(node, list):
        for v in node:
            walk(v, out)


def main():
    try:
        data = json.load(sys.stdin)
    except json.JSONDecodeError as e:
        print(f"extract.py: failed to parse JSON: {e}", file=sys.stderr)
        sys.exit(2)
    out = []
    walk(data, out)
    for cmd in out:
        print(cmd)


if __name__ == "__main__":
    main()


# EOF
