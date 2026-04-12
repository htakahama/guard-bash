# Configuration

guard-bash の動作は TOML 設定ファイルと環境変数で調整できる。

## 検索順序

設定ファイルは以下の順で検索される。最初に見つかったものを読み込む:

1. `$GUARD_CONFIG` (絶対パス)
1. `$XDG_CONFIG_HOME/guard-bash/config.toml`
1. `$HOME/.config/guard-bash/config.toml`

ファイルが存在しない場合は embedded default のみで動作する。

## TOML schema

```toml
[policy]
# allowed / denied を非空で指定すると embedded default を完全に置き換える。
allowed = ["git", "gh", "cat", ...]
denied  = ["sudo", "eval", ...]

# extra_allowed / extra_denied は embedded default に追加する。
extra_allowed = ["my-tool"]
extra_denied  = ["curl"]

[checkcd]
# cwd 以外に cd が許可される dir (サブディレクトリも含む)
allowed_dirs = ["/home/user/work", "/tmp/scratch"]

[logging]
# "debug" | "info" | "warn" | "error"
level = "info"
# 空なら ${XDG_STATE_HOME:-$HOME/.local/state}/guard-bash/guard-bash.log
file  = ""
```

## 環境変数オーバーライド

| 変数                  | 型         | 効果                                                            |
| --------------------- | ---------- | --------------------------------------------------------------- |
| `GUARD_CONFIG`        | path       | TOML の読み込み先を明示                                         |
| `GUARD_EXTRA_ALLOWED` | `:` 区切り | `policy.extra_allowed` に追加                                   |
| `GUARD_EXTRA_DENIED`  | `:` 区切り | `policy.extra_denied` に追加                                    |
| `GUARD_ALLOWED_DIRS`  | `:` 区切り | `checkcd.allowed_dirs` に追加 (Claude Code の `--add-dir` 相当) |
| `GUARD_LOG_LEVEL`     | string     | `logging.level` を上書き                                        |
| `GUARD_LOG_FILE`      | path       | `logging.file` を上書き                                         |

## allowed / denied の優先順位

1. `denied` (default) / `extra_denied` に一致 -> ブロック
1. `allowed` (default) + `extra_allowed` の union に含まれていない -> ブロック
1. `extra_denied` に含まれる名前は `allowed`/`extra_allowed` から取り除かれる

動的コマンド名 (`$cmd` など) は常にブロックされる。

## ログフォーマット

`log/slog` JSON handler が 1 呼び出し 1 行で出力する。

```json
{
  "time": "2026-04-12T00:51:55.914827155Z",
  "level": "INFO",
  "msg": "allow",
  "cwd": "/home/user/work/ghq/guard-bash",
  "cmd": "for f in $(git ls-files | head); do cat \"$f\"; done",
  "extracted": ["git", "head", "cat"],
  "fixed_cmd": "cd /home/user/work/ghq/guard-bash && for f in ...",
  "duration_ms": 4
}
```

block 時は `level=WARN`, `msg=deny`, `err` フィールドに理由が入る。

> [!NOTE]
> ログローテーションは guard-bash 本体では行わない。logrotate や
> journald 側でハンドルする想定。

<!-- EOF -->
