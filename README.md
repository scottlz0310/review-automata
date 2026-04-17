# review-automata

[![CI](https://github.com/scottlz0310/review-automata/actions/workflows/ci.yml/badge.svg)](https://github.com/scottlz0310/review-automata/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/scottlz0310/review-automata/branch/main/graph/badge.svg)](https://codecov.io/gh/scottlz0310/review-automata)
[![Go Version](https://img.shields.io/badge/go-1.26.2-blue.svg)](https://go.dev/)

Copilot レビュー完了通知メールをトリガーとして、ローカルの Claude CLI による自動対応処理を実行する **メール駆動型 Go CLI ツール**。

```
Copilot レビュー完了メール
  → subject パース（owner/repo/PR番号 抽出）
  → ~/src 配下のリポジトリ特定
  → PR ブランチを fetch & checkout
  → Claude CLI にレビューコメントを STDIN で渡す
  → 自動修正コミット・ループ継続
```

---

## 目次

- [動作要件](#動作要件)
- [セットアップ（ユーザー向け）](#セットアップユーザー向け)
- [使い方](#使い方)
- [設定リファレンス](#設定リファレンス)
- [アーキテクチャ](#アーキテクチャ)
- [開発者向け](#開発者向け)
- [ロードマップ](#ロードマップ)

---

## 動作要件

| 要件 | バージョン / 条件 |
|------|-----------------|
| Go | 1.26.2 以上 |
| OS | Windows（`tasklist` / `taskkill` に依存） |
| Gmail | IMAP アクセス有効化 + App Password 発行済み |
| [Claude CLI](https://docs.anthropic.com/en/docs/claude-code) | `claude` コマンドがパス上に存在すること |
| [gh CLI](https://cli.github.com/) | v0.6.0 以降で使用予定（現時点は任意） |
| リポジトリ配置 | `~/src/<repo名>/` に clone されていること |

---

## セットアップ（ユーザー向け）

### 1. リポジトリをクローン

```bash
git clone https://github.com/scottlz0310/review-automata.git
cd review-automata
```

### 2. .env を作成する

```bash
cp sample.env .env
```

`.env` を開き、以下を設定してください。

```dotenv
MAIL_IMAP_ADDR=imap.gmail.com:993   # Gmail のデフォルト値のまま可
MAIL_USERNAME=yourname@gmail.com    # Gmail アドレス
MAIL_PASSWORD=xxxxxxxxxxxxxxxx      # Gmail App Password（16 文字・スペースなし）
MAIL_MAILBOX=INBOX                  # 監視するメールボックス
```

**Gmail App Password の取得方法**:  
Google アカウント → セキュリティ → 2段階認証を有効化 → アプリパスワードを作成

> `.env` はコミット禁止です（`.gitignore` で除外済み）。

### 3. ビルドして起動

```bash
go build -o review-automata.exe ./cmd/review-automata/
./review-automata.exe
```

もしくはビルドなしで直接起動:

```bash
go run ./cmd/review-automata/
```

---

## 使い方

### 起動

```bash
./review-automata.exe
# 情報: review-automata を起動しました (IMAP: imap.gmail.com:993, Mailbox: INBOX)
```

起動後は IMAP IDLE でメールボックスを監視します。Copilot レビュー完了通知メールが届くと自動的に処理を開始します。

### 終了

`Ctrl+C` で安全に終了します。

### 処理フロー

1. Copilot が GitHub PR レビューを完了すると通知メールが届く
2. subject の正規表現パースで `owner` / `repo` / `PR番号` を抽出
3. `~/src/` 配下で対象リポジトリを特定し、origin URL で照合
4. `git fetch origin pull/{PR}/head:pr-{PR}` + `git checkout pr-{PR}`
5. レビューコメント本文をプロンプトとして構築し、Claude CLI へ STDIN 経由で渡す

### STOP 条件（何もしないで次のメールを待つ）

以下の条件に該当すると処理を中断し、エラーをログに出力して次のメールを待ちます。

| 条件 | 説明 |
|------|------|
| subject パース失敗 | Copilot 通知メールのフォーマット外 |
| リポジトリ未検出 | `~/src/` 配下に対象リポジトリが見つからない |
| リポジトリ複数検出 | 同名リポジトリが複数存在する |
| origin 不一致 | `git remote get-url origin` が期待値と異なる |
| PR checkout 失敗 | `git fetch` または `git checkout` が失敗 |

---

## 設定リファレンス

| 環境変数 | デフォルト値 | 説明 |
|---------|------------|------|
| `MAIL_IMAP_ADDR` | `imap.gmail.com:993` | IMAP サーバーアドレス（TLS） |
| `MAIL_USERNAME` | （必須） | メールアドレス |
| `MAIL_PASSWORD` | （必須） | App Password（OAuth2 は v1.0.0 予定） |
| `MAIL_MAILBOX` | `INBOX` | 監視するメールボックス名 |

---

## アーキテクチャ

```
cmd/
  review-automata/      ← CLI エントリポイント・ループ制御
internal/
  mail/                 ← IMAP IDLE によるメール監視（PoC: go-imap + App Password）
  parser/               ← subject / 本文パース、STOP 判定
  resolver/             ← ~/src 配下のリポジトリ特定・origin URL 照合
  git/                  ← PR ブランチ fetch + checkout
  executor/             ← プロセス管理・プロンプト構築・Claude CLI 委譲（STDIN）
docs/
  design.md             ← 基本設計サマリ
  roadmap.md            ← リリース計画
```

**メール監視の移行計画**:

| フェーズ | 方式 | 認証 |
|---------|------|------|
| PoC（現在: v0.4.0〜） | IMAP IDLE（`go-imap`） | Gmail App Password |
| 本実装（v1.0.0 予定） | Gmail API + Cloud Pub/Sub | OAuth2 |

---

## 開発者向け

### 開発環境のセットアップ

**必要なツール**:
- Go 1.26.2 以上
- [golangci-lint](https://golangci-lint.run/usage/install/)
- [lefthook](https://github.com/evilmartians/lefthook) （pre-commit フック管理）

**セットアップ手順**:

```bash
git clone https://github.com/scottlz0310/review-automata.git
cd review-automata

# 依存解決
go mod download

# pre-commit / pre-push フックを登録
lefthook install
```

### ビルド・テスト

```bash
# 全体ビルド
go build ./...

# CLI バイナリのみビルド
go build ./cmd/review-automata/

# 全テスト
go test ./...

# レース検出 + カバレッジ計測
go test -v -race -coverprofile=coverage.out ./...

# 特定パッケージのみ
go test -run TestParser ./internal/parser/

# Lint
golangci-lint run ./...
```

### pre-commit フック（lefthook）

`lefthook install` を実行すると、コミット・プッシュ時に以下が自動実行されます。

| タイミング | チェック内容 |
|-----------|------------|
| `pre-commit` | `gofmt` フォーマット確認、`go vet`、`go build` |
| `pre-push` | `main` ブランチへの直接 push を禁止 |

### ブランチ戦略

```bash
# 作業ブランチを作成
git checkout -b feature/v{X.Y.Z}-{概要}

# ... 実装・テスト・lint ...

# コミット（Conventional Commits 形式）
git commit -m "feat: {概要}" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"

git push origin feature/v{X.Y.Z}-{概要}
```

**コミットプレフィックス**:

| プレフィックス | 用途 |
|-------------|------|
| `feat:` | 新機能 |
| `fix:` | バグ修正 |
| `docs:` | ドキュメント |
| `refactor:` | リファクタリング |
| `test:` | テスト |
| `chore:` | ビルド・依存・設定 |
| `ci:` | CI/CD |

**PR サイズ目安: 〜300 行**

### テスト規約

- テーブル駆動テスト（`t.Run`）を基本とする
- 外部依存（IMAP・Git・Claude CLI）はインターフェース経由でモックする
- 各テストに目的コメント: `// 検証対象: XXX  目的: YYY`

### パッケージ設計方針

- 各 `internal/` パッケージは単一責務
- パッケージ間の循環 import 禁止
- エラーメッセージは日本語、原因が分かる内容にする
- ログは `os.Stderr`。機密情報（パスワード・トークン）はログに出力しない

---

## ロードマップ

| バージョン | 状態 | 内容 |
|-----------|------|------|
| v0.1.0 | ✅ リリース済み | プロジェクト基盤・CI/CD |
| v0.2.0 | ✅ リリース済み | メール subject / 本文パーサー |
| v0.3.0 | ✅ リリース済み | リポジトリ特定・PR ブランチ checkout |
| v0.4.0 | ✅ リリース済み | IMAP IDLE 統合・E2E 動作確認（Windows） |
| v0.5.0 | 🚧 開発中 | Claude CLI Executor（プロンプト構築・委譲） |
| v0.6.0 | 📋 予定 | Copilot 再レビューリクエスト自動化（gh CLI） |
| v1.0.0 | 📋 予定 | Gmail API + Cloud Pub/Sub・本番品質移行 |

詳細は [docs/roadmap.md](docs/roadmap.md) を参照してください。
