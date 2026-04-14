# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- Go プロジェクト構造（`cmd/review-automata/`, `internal/` パッケージ群）
- `go.mod`（module: `github.com/scottlz0310/review-automata`）
- GitHub Actions CI（golangci-lint + go test -race + codecov）
- pre-commit フック（go-fmt / go-vet / go-build / no-commit-to-branch）
- `codecov.yml`
- `docs/roadmap.md`（PoC: IMAP IDLE → 本実装: Gmail API + Pub/Sub の2段構成）
- `tasks.md`, `docs/design.md`
