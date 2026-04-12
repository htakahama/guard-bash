# guard-bash

Claude Code の `PreToolUse` フックとして Bash ツール呼び出しを検証する
単一バイナリ。`mvdan.cc/sh/v3/syntax` で Bash AST を解析し、`for` / `while` /
`if` / パイプ / コマンド置換など複雑な構文の全コマンド呼び出しを
allowlist / denylist と突合する。

## 特徴

- 単一バイナリ。`bash` / `python` / `jq` / `shfmt` に依存しない
- AST ベース検査: `for f in $(git ls-files); do rm "$f"; done` のような
  ネストされた構文でも内部の `rm` を正しく検出
- 先頭 `cd <path>` の AST ベース判定と許可 dir 配下チェック
- TOML 設定 (embedded default + ユーザー override)
- 構造化ログ (`log/slog` JSON handler) をファイル出力

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
> フルパス指定の場合は `$HOME/go/bin/guard-bash` (go install) や
> `mise where github:htakahama/guard-bash` の出力を使う。

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
