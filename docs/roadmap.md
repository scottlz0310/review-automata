# Roadmap

基本設計: [Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38)

---

## v0.1.0 — Project Bootstrap

**目標**: CI/CD・開発基盤の整備（Go）

- [x] Go プロジェクト構造（`cmd/` + `internal/`）
- [x] GitHub Actions CI (golangci-lint + go test + codecov)
- [x] pre-commit フック（main直push禁止含む）
- [x] CHANGELOG / tasks.md / docs/roadmap.md

---

## v0.2.0 — Mail Parser

**目標**: メールの subject / 本文から PR 情報を抽出

- subject 正規表現パース（owner / repo / PR番号）
- 本文クリーニング（不要ヘッダ/フッタ除去）
- STOP 条件: パース失敗

---

## v0.3.0 — Repository Resolver + Git Operations

**目標**: リポジトリ特定と PR ブランチ checkout

- `~/src` 配下で repo 名 + origin URL による一致確認
- `git fetch origin pull/{PR}/head:pr-{PR}` + checkout
- 既存ブランチへの影響ゼロ保証
- STOP 条件: 0件 / 複数件 / checkout 失敗

---

## v0.4.0 — PoC: IMAP IDLE Integration

**目標**: IMAP IDLE によるメール監視 + エンドツーエンド動作確認

- `go-imap` + IDLE でリアルタイムメール受信
- App Password 認証（GCP 不要）
- v0.2/v0.3 との統合
- Windows 上での動作確認

---

## v0.5.0 — Claude CLI Executor

**目標**: プロンプト構築と Claude CLI への委譲

- プロンプトテンプレート実装
- STDIN 経由での CLI 実行
- 終了コード・エラーハンドリング

---

## v0.6.0 — Review Loop: Copilot 再レビューリクエスト

**目標**: 修正後の Copilot 再レビュー依頼を自動化し、ループを完結させる

- `gh pr edit --add-reviewer "copilot-pull-request-reviewer[bot]"` による再依頼
- レビュー完了通知メールによるループ継続判定（メール確定原則）
- ループ終了条件の実装（blocking コメント 0 件 + CI SUCCESS）
- REST API 非使用（[Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38) の実証結果に基づく）

---

## v1.0.0 — 本実装: Gmail API + Cloud Pub/Sub

**目標**: PoC から本番品質への移行

- Gmail API + OAuth2 認証
- Cloud Pub/Sub によるプッシュ通知
- Windows Service 化（`golang.org/x/sys/windows/svc`）
- E2E テスト・ドキュメント整備

---

## 将来検討

- 複数メールプロバイダー対応
- Web UI / 設定ファイル管理
