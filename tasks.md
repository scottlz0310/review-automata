# Tasks

進行中・予定タスクを管理します。完了したものは CHANGELOG へ移動。

Epic ISSUE は GitHub の `scottlz0310/review-automata` で管理。

| リリース | Epic |
|---------|------|
| v0.1.0 | [#1](https://github.com/scottlz0310/review-automata/issues/1) |
| v0.2.0 | [#2](https://github.com/scottlz0310/review-automata/issues/2) |
| v0.3.0 | [#3](https://github.com/scottlz0310/review-automata/issues/3) |
| v0.4.0 | [#4](https://github.com/scottlz0310/review-automata/issues/4) |
| v0.5.0 | [#5](https://github.com/scottlz0310/review-automata/issues/5) |
| v1.0.0 | [#6](https://github.com/scottlz0310/review-automata/issues/6) |

---

## v0.1.0 — Project Bootstrap

Epic: [#1](https://github.com/scottlz0310/review-automata/issues/1)

| # | Sub-Issue | タスク | 状態 |
|---|-----------|--------|------|
| 1 | - | Go プロジェクト構造（go.mod + cmd/ + internal/） | ✅ |
| 2 | - | GitHub Actions CI | ✅ |
| 3 | - | pre-commit 設定 | ✅ |
| 4 | - | codecov.yml | ✅ |
| 5 | - | CHANGELOG / tasks.md / docs/ | ✅ |
| 6 | - | Epic ISSUE 作成 (v0.2.0〜v1.0.0) | ✅ |
| 7 | - | codecov トークン設定 (repo secret) | 🔲 |
| 8 | - | PR review cycle ルール (他 repo から移植) | ✅ |
| 9 | - | `go mod tidy` + 初期依存解決 | 🔲 |

---

## v0.2.0 — Mail Parser

Epic: [#2](https://github.com/scottlz0310/review-automata/issues/2)

| # | Sub-Issue | タスク | 状態 |
|---|-----------|--------|------|
| 1 | - | subject 正規表現パース（owner / repo / PR番号） | ✅ |
| 2 | - | 本文クリーニング（不要ヘッダ/フッタ除去） | ✅ |
| 3 | - | STOP 条件: パース失敗時は error を返す | ✅ |
| 4 | - | ユニットテスト（正常系 4 / 異常系 5） | ✅ |
| 5 | - | codecov カバレッジ通過 | 🔲 CI確認待ち |

---

## v0.3.0 — Repository Resolver + Git Operations

Epic: [#3](https://github.com/scottlz0310/review-automata/issues/3)

| # | Sub-Issue | タスク | 状態 |
|---|-----------|--------|------|
| 1 | - | repo 名で候補検索（`~/src` 配下） | ✅ |
| 2 | - | origin URL による一致確認 | ✅ |
| 3 | - | STOP 条件: 0件 / 複数件 / origin 不一致 | ✅ |
| 4 | - | `git fetch origin pull/{PR}/head:pr-{PR}` + `git checkout pr-{PR}` | ✅ |
| 5 | - | STOP 条件: checkout 失敗 | ✅ |
| 6 | - | 既存ブランチへの影響ゼロ保証 | ✅ |
| 7 | - | ユニットテスト | ✅ |

---

## v0.4.0 以降

詳細設計後に展開。
