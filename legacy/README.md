# guard-bash

Claude Code の PreToolUse フックとして Bash ツール呼び出しを検証する。

`shfmt -tojson` で Bash AST を取得し、`for` / `while` / `if` / パイプ /
コマンド置換で呼び出される全コマンドを whitelist / denylist と突合する。

## 構成

```text
guard-bash/
├── main.sh       # エントリポイント (Claude Code から起動)
├── extract.py    # AST から CallExpr のコマンド名を抽出
├── check_cd.py   # 先頭 cd のターゲットパスを許可 dir 配下か検査
├── test.sh       # 回帰テスト
└── README.md
```

## 検査フロー

1. stdin の JSON を jq で分解 (`.cwd`, `.tool_input.command`, `.tool_input.description`)
2. `cwd` が git 管理下か確認
3. `shfmt -tojson` で AST を生成。パース失敗は fail-closed でブロック
4. `extract.py` で CallExpr の全コマンド名を収集 (動的語は `__DYNAMIC__`)
5. denylist / allowlist と突合。1 つでも NG ならブロック
6. `check_cd.py` で先頭 stmt の cd を判定:
    - 先頭が `cd <絶対パス>` で許可 dir 配下 -> そのまま allow
    - 先頭が `cd` でない -> `cd $CWD && ...` を自動付与して allow
    - 先頭が `cd` で許可 dir 配下外 or 動的パス -> ブロック
7. `updatedInput` を JSON で stdout に出力

## 許可 dir

`check_cd.py` は以下を許可 dir とみなす:

- `GUARD_CWD` 環境変数 (Claude Code hook 入力の `.cwd`)
- `GUARD_ALLOWED_DIRS` 環境変数 (`:` 区切り、Claude Code の `--add-dir` 相当)

## ラッパーコマンド

`env` / `command` / `nice` / `nohup` は先頭に来た場合、続く非代入・非フラグ
引数も追加でコマンド名として抽出する。以下は単純化による既知の制約:

- `env -u VAR CMD` のような `-flag ARG` 形式は正しく解釈できない
- `command -v git` のようなビルトインオプションで誤検出する可能性がある

`time` は shfmt が `TimeClause` として別 AST ノードに切り出すため、
走査で自然に内側の CallExpr を拾える。

## テスト

```bash
./test.sh
```

15 ケースの正常系 / 異常系を網羅。`shfmt` が PATH にあることが前提。

## 配備

dotfiles 側の `claude/{default,personal}/settings.json` の `PreToolUse`
フックで `main.sh` を絶対パス指定で呼び出す。


<!-- EOF -->
