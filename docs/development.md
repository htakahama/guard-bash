# Development

mise で tool / task を管理する。最初に `mise trust` と `mise install` を実行して
Go 1.26 / golangci-lint / goreleaser / rumdl を揃える。

## セットアップ

```bash
cd ~/work/ghq/guard-bash
mise trust
mise install
```

## 日常の開発ループ

```bash
mise run fmt       # gofmt -w .
mise run vet       # go vet ./...
mise run test      # go test ./...
mise run check     # fmt:check + vet + lint + test + md:lint + md:fmt:check
```

`check` は CI でも同じタスクを走らせる。push 前に必ず green にする。

## テスト

ユニットテスト (white-box):

```bash
mise run test
```

Integration テスト (blackbox, 実バイナリを子プロセスで spawn する):

```bash
mise run test:integration
```

Integration テストは `test/integration/integration_test.go` に集約され、`//go:build integration`
タグで通常のテストから分離されている。
各ケースは`t.TempDir()` + `git init` でサンドボックス化しているため並列実行安全。

## ビルド

```bash
mise run build            # dist/guard-bash (host platform)
mise run install          # go install -> $GOBIN/guard-bash
mise run release:snapshot # goreleaser で全 platform をビルド (publish なし)
```

snapshot ビルドの成果物は `dist/` 配下に出力され `.gitignore` で除外。

## リリース

1. 変更を main にマージ
1. `git tag vX.Y.Z && git push origin vX.Y.Z`
1. `.github/workflows/release.yml` の GoReleaser job が起動し、GitHub Releases にアーティファクトを添付する

タグは [Semantic Versioning](https://semver.org/) に従う。
changelog は Conventional Commits の `feat` / `fix` / `refactor` / `perf` のみ抽出してリリースノートに掲載する。

## コード規約

- `gofmt -w` を必ず通す (`mise run fmt`)
- `golangci-lint` 設定は `.golangci.yml` (リポ内、デフォルト lint を使用)
- エラーは `fmt.Errorf(...%w...)` で wrap
- パッケージ冒頭に 1 段落の doc コメントを書く
- 行末コメントや冗長なインラインコメントは書かない
- テストファイルの末尾には `// EOF` を付記する (dotfiles ルール準拠)

## ディレクトリ

```text
guard-bash/
├── cmd/guard-bash/main.go         # エントリ
├── internal/
│   ├── config/                    # TOML + env
│   ├── hook/                      # Claude Code JSON I/O
│   ├── parse/                     # sh/v3 ラッパ + Word/Cmd ユーティリティ
│   ├── extract/                   # CallExpr 走査
│   ├── checkcd/                   # cd 許可判定
│   ├── policy/                    # allow/deny 突合
│   └── logging/                   # slog 初期化
├── test/integration/              # blackbox テスト
├── docs/                          # 本ドキュメント群
├── legacy/                        # 旧 bash+python 実装 (参考用)
└── dist/                          # goreleaser 出力 (.gitignore)
```

<!-- EOF -->
