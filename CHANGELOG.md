# Changelog

このプロジェクトの重要な変更はすべてこのファイルに記録されます。

このフォーマットは [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) に基づいており、
このプロジェクトは [Semantic Versioning](https://semver.org/spec/v2.0.0.html) に準拠しています。

---

## [Unreleased]

### Added
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
