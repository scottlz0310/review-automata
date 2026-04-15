# 基本設計

基本設計は [Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38) に記載。

技術的背景・根拠は [Mcp-Docker#47](https://github.com/scottlz0310/Mcp-Docker/issues/47) を参照。

---

## アーキテクチャ概要

### 単発フロー（メール受信 → Claude 対応）

```
メール（Copilotレビュー完了通知）
  ↓
パース（subject中心）
  ↓
ローカルエージェント
  ↓
Delegate（実行主体）
  ↓
CLI（実行エンジン）
```

### 自動ループ（継続的レビュー対応）

```
修正コミット・プッシュ
  ↓
gh review request（トリガー）
  ↓
Copilot レビュー実行
  ↓
レビュー完了通知メール（確定イベント）
  ↓
メール受信・パース
  ↓
自動対応（Claude CLI 実行）
  ↓
（必要に応じて繰り返し）
```

---

## モジュール構成（予定）

設計上の論理名と実装上のパッケージ名は分けて管理する。

| 論理名 | 実装パッケージ / ディレクトリ | 責務 |
|--------|------------------------------|------|
| `mail_parser` | `internal/parser` | subject/本文パース、STOP判定 |
| `repo_resolver` | `internal/resolver` | ~/src からリポジトリ特定 |
| `git_ops` | `internal/git` | PR fetch & checkout |
| `executor` | `internal/executor` | プロンプト構築・Claude CLI 呼び出し |
| `reviewer` | `internal/reviewer` | gh CLI 経由の Copilot レビューリクエスト |
| `cli` | `cmd/review-automata` | エントリポイント・ループ制御 |

---

## Copilot レビューリクエスト方針

[Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38) での実証結果に基づく。

### 採用手段

```bash
gh pr edit <PR番号> --add-reviewer "copilot-pull-request-reviewer[bot]"
```

### REST API との比較

| 項目 | REST API（旧） | gh CLI（採用） |
|------|--------------|--------------|
| 成功判定 | 不正確（偽成功あり） | 実処理ベース |
| イベント発火 | 非決定的 | 発火確認済み |
| 再現性 | 低い | 現時点で良好 |

### 成功判定の原則

- `gh` コマンドの応答は「トリガー成功」に過ぎない
- **最終的な成功判定はレビュー完了通知メールの受信をもって行う**
- `gh` の内部実装は非公開のため、メール駆動による検証フローは常に維持する

---

## STOP 条件

- subject パース失敗
- repo 未検出 / 複数
- origin 不一致
- PR checkout 失敗

---

## 設計方針

| 原則 | 内容 |
|------|------|
| 安全優先 | 曖昧な場合は何もしない |
| 環境保護 | `~/src` 既存ブランチを変更しない |
| シンプル | メール本文は構造化しない |
| 責任分離 | CLI は実行エンジン、Delegate が実行責任を持つ |
| イベント性 | メールは「イベント」であり「命令」ではない |
| メール確定 | `gh` レスポンスでなくメール受信を成功の確定イベントとする |
