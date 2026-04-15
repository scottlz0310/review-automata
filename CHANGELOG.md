# Changelog

このプロジェクトの重要な変更はすべてこのファイルに記録されます。

このフォーマットは [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) に基づいており、
このプロジェクトは [Semantic Versioning](https://semver.org/spec/v2.0.0.html) に準拠しています。

---

## [Unreleased]

### Added
- `internal/reviewer` モジュール追加（gh CLI 経由の Copilot 再レビューリクエスト）
- `docs/roadmap.md` に v0.6.0 フェーズを追加（Review Loop: Copilot 再レビュー自動化）

### Changed
- `docs/design.md`: 自動ループフロー・reviewer モジュール・Copilot レビューリクエスト方針を追記
- `.github/copilot-instructions.md`: reviewer モジュールをアーキテクチャに追加、再レビュー依頼コマンドを明記
- 設計方針に「メール確定」原則を追加（gh レスポンスでなくメール受信を成功の確定イベントとする）

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
