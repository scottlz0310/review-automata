# Changelog

このプロジェクトの重要な変更はすべてこのファイルに記録されます。

このフォーマットは [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) に基づいており、
このプロジェクトは [Semantic Versioning](https://semver.org/spec/v2.0.0.html) に準拠しています。

---

## [Unreleased]

---

## [0.5.0] - 2026-04-17 (Unreleased)

### Added
- `internal/executor`: `ProcessManager` インターフェース — `IsRunning` / `Kill` を抽象化（外部依存モック可能）
- `internal/executor`: `CLIRunner` インターフェース — `RunWithStdin` を抽象化
- `internal/executor`: `ExecProcessManager` — `tasklist` / `taskkill` を使う実装
- `internal/executor`: `ExecCLIRunner` — stdin パイプで `claude` CLI を起動する実装、非ゼロ終了コードはエラー返却
- `internal/executor`: `Executor` 構造体（`New` / `IsAgentRunning` / `KillAgent` / `BuildPrompt` / `Run`）
- `internal/executor`: `BuildPrompt` — Mcp-Docker#38 仕様準拠のプロンプトテンプレート生成
- `internal/executor/executor_test.go`: テーブル駆動テスト 11 ケース（BuildPrompt 3 / IsAgentRunning 3 / KillAgent 2 / Run 3）

### Changed
- `cmd/review-automata/main.go`: `executor.New(ExecProcessManager{}, ExecCLIRunner{})` で `Executor` を構築し `Run()` を呼び出すよう変更（E2E 統合完成）
- `tasks.md`: v0.5.0 タスクセクション追加・全完了マーク

---

## [0.4.1] - 2026-04-17

### Added
- `internal/git`: `ErrBranchExists` sentinel error（`errors.Is` で判定可能）
- `internal/git`: `ForceUpdate` — `checkout main` → `fetch --force` → `checkout pr-N` で既存ブランチを強制更新
- `cmd/review-automata/main.go`: `ErrBranchExists` 検出時に `[y/N]` 確認プロンプトを常時表示し、承認後に `ForceUpdate` を実行するフロー

### Changed
- `internal/executor`: `IsAgentRunning()` を `(bool, error)` に変更。判定不能時はエラーを返す
- `internal/executor`: `KillAgent()` を修正。プロセス未存在は許容、`taskkill` 実行失敗時はエラーを返す
- `internal/git`: `ErrBranchExists` ラップ時のエラーメッセージを `"ブランチ %q が既に存在します: %w"` に修正
- `internal/git/git_test.go`: `TestFetchAndCheckout_ErrBranchExists` / `TestForceUpdate`（6ケース）追加

---

## [0.4.0] - 2026-04-17

### Added
- `internal/mail`: `Watcher` — IMAP IDLE ベースのメール監視（PoC）。`go-imap` v1 + App Password 認証
- `internal/mail`: `Config` / `MessageHandler` 型、`New` / `Watch` 関数
- `internal/mail`: `extractTextBody` — MIME メッセージから text/plain 部分を抽出（`go-message` 使用）
- `internal/mail/mail_test.go`: テーブル駆動テスト（Config バリデーション 6 件 / Watch 無効設定 2 件 / extractTextBody 3 件）
- `cmd/review-automata/main.go`: E2E エントリポイント（メール監視 → パース → リポジトリ解決 → checkout の統合フロー）
- `sample.env`: IMAP 接続設定テンプレート（`MAIL_IMAP_ADDR` / `MAIL_USERNAME` / `MAIL_PASSWORD` / `MAIL_MAILBOX`）
- `.gitignore`: `.env` をコミット対象外に追加

### Changed
- `go.mod` / `go.sum`: `github.com/emersion/go-imap` v1.2.1、`github.com/emersion/go-message` v0.18.2、`github.com/joho/godotenv` v1.5.1 を追加

---

## [0.3.0] - 2026-04-15

### Added
- `internal/resolver`: `Resolver.Resolve` — `~/src` 配下から owner/repo に一致するリポジトリを特定（STOP条件: 0件/複数件/origin不一致）
- `internal/resolver`: `GitRunner` インターフェース + `ExecGitRunner` 実装
- `internal/resolver/resolver_test.go`: テーブル駆動テスト（正常系2件 + STOP 7件 + URL バリアント 8件）
- `internal/git`: `FetchAndCheckout` — `git fetch origin pull/{N}/head:pr-{N}` + `git checkout pr-{N}`（STOP条件: dir未指定 / PR番号不正 / Commander未設定 / 既存ブランチ検出 / fetch/checkout 失敗）
- `internal/git`: `Commander` インターフェース + `ExecCommander` 実装
- `internal/git/git_test.go`: テーブル駆動テスト（正常系1件 + STOP 6件）
- `internal/parser`: `ParseSubject` — subject から owner/repo/PR番号を抽出（STOP条件: パース失敗）
- `internal/parser`: `CleanBody` — GitHub 通知メールの不要フッター除去
- `internal/parser/parser_test.go`: テーブル駆動テスト（正常系 4 / 異常系 5 の計 15 ケース）

### Changed
- `docs/design.md`: 自動ループフロー・reviewer モジュール（予定）・Copilot レビューリクエスト方針・メール確定原則を追記
- `docs/roadmap.md`: v0.6.0 フェーズ（Review Loop: Copilot 再レビュー自動化）を追加
- `.github/copilot-instructions.md`: reviewer モジュール（予定）をアーキテクチャに追加、再レビュー依頼コマンドを明記

---

## [0.1.0] - 2026-04-15

### Added
- Go プロジェクト構造（`cmd/review-automata/`, `internal/` パッケージ群）
- `go.mod`（module: `github.com/scottlz0310/review-automata`）
- GitHub Actions CI（golangci-lint + go test -race + codecov）
- pre-push フック（go-fmt / go-vet / go-build / no-push-to-main）
- `codecov.yml`
- `docs/roadmap.md`（PoC: IMAP IDLE → 本実装: Gmail API + Pub/Sub の2段構成）
- `tasks.md`, `docs/design.md`
