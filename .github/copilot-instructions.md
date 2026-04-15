# Copilot Instructions

## 言語規約
**すべてのコード、コメント、ドキュメント、AI との対話は日本語で行う。** 技術的固有名詞（パッケージ名・API 名等）は英語併記可。

## プロジェクト概要
Copilot レビュー完了通知メールをトリガとして、ローカル Claude CLI による自動対応処理を実行するメール駆動型 Go CLI ツール。

アーキテクチャ詳細・設計根拠は [Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38) を参照。

## ビルド・テスト

```bash
go build ./...                          # 全体ビルド
go build ./cmd/review-automata/         # CLI バイナリのみ
go test ./...                           # 全テスト
go test -v -race ./...                  # レース検出付き
go test -run TestParser ./internal/parser/   # パッケージ絞り込み
go test -v -race -coverprofile=coverage.out ./...  # カバレッジ計測
golangci-lint run ./...                 # lint
lefthook install                        # コミットフック登録（初回セットアップ時）
```

## アーキテクチャ

```
cmd/
  review-automata/      ← CLI エントリーポイント・ループ制御（cobra 導入予定）
internal/
  mail/                 ← メール監視（PoC: IMAP IDLE / 本実装: Gmail API + Pub/Sub）
  parser/               ← subject / 本文パース、STOP 判定
  resolver/             ← ~/src 配下のリポジトリ特定
  git/                  ← PR fetch + checkout
  executor/             ← プロンプト構築・Claude CLI 委譲（STDIN）
  reviewer/             ← 予定 (v0.6.0): gh CLI 経由の Copilot 再レビューリクエスト
docs/
  design.md             ← 基本設計サマリ（参照元: Mcp-Docker#38）
  roadmap.md            ← リリース計画
```

### メール監視の2段構成

| フェーズ | 方式 | 認証 |
|---------|------|------|
| PoC (v0.4.0) | IMAP IDLE（`go-imap`） | Gmail App Password |
| 本実装 (v1.0.0) | Gmail API + Cloud Pub/Sub | OAuth2 |

### STOP 条件（設計原則: 曖昧な場合は何もしない）
- subject パース失敗
- リポジトリ未検出 / 複数
- origin URL 不一致
- PR checkout 失敗

## 重要な規約

### パッケージ設計
- 各 `internal/` パッケージは単一責務。パッケージ間の循環 import 禁止。
- `internal/` 外への公開 API は最小限にする。

### エラーハンドリング
- STOP 条件に該当する場合は `fmt.Errorf` でラップして呼び出し元に返す。自律的にリカバリしない。
- エラーメッセージは日本語で、原因が分かる内容にする。

### ログ
- 標準エラー出力（`os.Stderr`）を使用。構造化ログは本実装フェーズで検討。
- 機密情報（パスワード・トークン）はログに出力しない。

### シークレット管理
- `.env` / 環境変数のみ（コミット禁止）。
- `sample.env` をテンプレートとして管理する。
- App Password・OAuth2 クレデンシャルをコードにハードコードしない。

### テスト規約
- テーブル駆動テスト（`t.Run`）を基本とする。
- 外部依存（IMAP・Git・Claude CLI）はインターフェース経由でモックする。
- 各テストに目的コメント: `// 検証対象: XXX  目的: YYY`

### コミット規約
Conventional Commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `ci:`
PR サイズ目安: ~300 行。

### タスク管理
現在の進捗は `tasks.md`（ルート）を参照。フェーズ完了時に更新すること。

---

## イテレーションサイクル

各フェーズは以下のサイクルで進める。

### 1. 実装
```bash
git checkout -b feature/v{X.Y.Z}-{名前}   # ブランチ作成
# ... コーディング ...
go build ./...                              # ローカルビルド確認
go test -v -race ./...                      # テスト確認
golangci-lint run ./...                     # lint 確認
```

### 2. CHANGELOG・tasks.md 更新
- `CHANGELOG.md` に当フェーズの変更内容を追記（Keep a Changelog 形式、日付入り）
- `tasks.md` の完了チェックボックスを更新

### 3. コミット & PR
```bash
git add -A
git commit -m "feat: {概要}" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
git push origin feature/v{X.Y.Z}-{名前}
gh pr create --title "v{X.Y.Z}: {タイトル}" --body "{説明}"
```

### 4. CI 監視（自動）
- **CI**: golangci-lint + go test -race + codecov が全 SUCCESS になるまで待機
- **Copilot code review**: PR 作成後に自動トリガー

### 5. レビュー対処（PR レビュー運用サイクル）

#### 基本ルール
- レビュー対応中は、対象 PR にのみコミットを積み、サブ PR を新規作成しない。
- `@copilot review` の再依頼は「前回レビューの指摘をすべて対応後」に 1 回だけ行う。
- 前回レビューが完了していない状態で新しいレビュー依頼を重ねて出さない。
- PR 作成後は **2 分間隔で最大 10 分**を目安に CI とレビューコメントを監視し、指摘があれば同一 PR で対応する。

#### サイクル終了条件
以下のいずれかを満たした場合、サイクルを終了する：
- `blocking` コメントがすべて解消されている
- 一定時間（2 分 × 3 回）新規コメントが発生していない（no comment）かつ CI が成功している

終了後、`non-blocking` / `suggestion` のみが残存する場合は、対応または対応見送り（理由明記）を行った上でサイクルを延長せず次工程へ進む。

#### ステップ 1: コメントを分類する

| 分類 | 基準 |
|------|------|
| `blocking`（修正必須） | 実行時エラーの可能性がある / データ整合性を破壊する / セキュリティリスク / 型安全性が欠如 / 後方互換性のない変更 |
| `non-blocking`（任意） | テスト追加・ログ改善など、対応推奨だが必須ではないもの |
| `suggestion`（改善提案） | 設計・命名・抽象化の改善提案 |

#### ステップ 2: 採否を判断する
- `accept` / `reject` を明示する。
- `reject` の場合は理由を記述する（スコープ外 / 別 PR で対応予定 / 設計方針と相違）。
- `non-blocking / suggestion` は「今回対応する」か「対応しない（理由明示）」のいずれかを選択する。

#### ステップ 3: 修正を行う
- `accept` した項目のみ修正する。修正後にビルド・テストを再実行する。

#### ステップ 4: 再レビューの要否を判断する

**再レビュー必須条件**
- 仕様・API 変更がある
- ロジック変更がある（分岐条件・計算式・データフローの変更）

**再レビュー不要条件**
- 軽微修正のみ（typo・コメント・フォーマット等）

#### ステップ 5: 最終アクションを決定する

- `blocking` が残存 → 再レビュー依頼
- `blocking` がすべて解消済み かつ 再レビュー必須 → 再レビュー依頼前に「前回指摘がすべて解消・新たな blocking なし」を確認してから依頼
- マージ条件をすべて満たす → **ユーザーにマージ許可を求める（自律的にマージしない）**

対処後：全スレッドに返信 → `resolveReviewThread` で全スレッドを解決済みに。

#### 再レビュー依頼コマンド

```bash
gh pr edit <PR番号> --add-reviewer "copilot-pull-request-reviewer[bot]"
```

> **注意**: REST API によるレビューリクエストは偽成功・イベント非発火の問題があるため使用しない。
> `gh` コマンドの応答は「トリガー成功」に過ぎず、**レビュー完了通知メールの受信をもって成功確定とする**。
> （根拠: [Mcp-Docker#38](https://github.com/scottlz0310/Mcp-Docker/issues/38)）

### 出力フォーマット
- コメント分類結果（blocking / non-blocking / suggestion の一覧）
- 採否一覧（accept / reject + reject 理由）
- 修正内容サマリ
- 再レビュー要否（理由付き）
- 最終アクション（再レビュー依頼 / マージ）

### 6. マージ条件（全て満たすこと）

> **⚠️ 重要: Copilot はユーザーから明示的に「マージしてください」と指示されない限り、自律的に PR をマージしてはならない。**
> マージ条件をすべて満たしていても、必ずユーザーに確認を求め、許可を得てからマージすること。

- [ ] ユーザーから明示的なマージ指示を受けている
- [ ] CI 全ジョブ SUCCESS（実行中なし）
- [ ] `blocking` コメント 0 件
- [ ] 未解決レビュースレッド 0 件
- [ ] 全コメントに返信済み
- [ ] 必要な承認数を満たしている（1 approval 以上）

### 7. クリーンアップ
```bash
git checkout main && git pull origin main
git branch -d feature/v{X.Y.Z}-{名前}
# リモートブランチは GitHub 側の "Delete branch" または PR マージ時自動削除
```
