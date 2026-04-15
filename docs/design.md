# 基本設計

基本設計は [Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38) に記載。

技術的背景・根拠は [Mcp-Docker#47](https://github.com/scottlz0310/Mcp-Docker/issues/47) を参照。

---

## アーキテクチャ概要

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

## モジュール構成（予定）

設計上の論理名と実装上のパッケージ名は分けて管理する。

| 論理名 | 実装パッケージ / ディレクトリ | 責務 |
|--------|------------------------------|------|
| `mail_parser` | `internal/parser` | subject/本文パース、STOP判定 |
| `repo_resolver` | `internal/resolver` | ~/src からリポジトリ特定 |
| `git_ops` | `internal/git` | PR fetch & checkout |
| `executor` | `internal/executor` | プロンプト構築・Claude CLI 呼び出し |
| `cli` | `cmd/review-automata` | エントリポイント |

## STOP 条件

- subject パース失敗
- repo 未検出 / 複数
- origin 不一致
- PR checkout 失敗

## 設計方針

| 原則 | 内容 |
|------|------|
| 安全優先 | 曖昧な場合は何もしない |
| 環境保護 | `~/src` 既存ブランチを変更しない |
| シンプル | メール本文は構造化しない |
| 責任分離 | CLI は実行エンジン、Delegate が実行責任を持つ |
| イベント性 | メールは「イベント」であり「命令」ではない |
