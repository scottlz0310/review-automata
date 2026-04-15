package parser

import (
	"testing"
)

// ParseSubject のテーブル駆動テスト
func TestParseSubject(t *testing.T) {
	// 検証対象: ParseSubject  目的: subject から owner/repo/PR番号を正しく抽出できること
	tests := []struct {
		name       string
		subject    string
		wantOwner  string
		wantRepo   string
		wantNumber int
		wantErr    bool
	}{
		{
			name:       "正常系: 標準フォーマット",
			subject:    "[scottlz0310/review-automata] Fix bug (PR #42)",
			wantOwner:  "scottlz0310",
			wantRepo:   "review-automata",
			wantNumber: 42,
		},
		{
			name:       "正常系: Re: プレフィックスあり",
			subject:    "Re: [owner/repo] Copilot review completed (PR #7)",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 7,
		},
		{
			name:       "正常系: PR番号が大きい",
			subject:    "[org/myrepo] Update dependencies (PR #1234)",
			wantOwner:  "org",
			wantRepo:   "myrepo",
			wantNumber: 1234,
		},
		{
			name:       "正常系: リポジトリ名にハイフン・ドットを含む",
			subject:    "[my-org/my.repo-name] Add feature (PR #99)",
			wantOwner:  "my-org",
			wantRepo:   "my.repo-name",
			wantNumber: 99,
		},
		{
			name:    "異常系: subject が空",
			subject: "",
			wantErr: true,
		},
		{
			name:    "異常系: [owner/repo] 形式なし",
			subject: "Copilot review completed PR #42",
			wantErr: true,
		},
		{
			name:    "異常系: PR番号なし",
			subject: "[owner/repo] Some review comment",
			wantErr: true,
		},
		{
			name:    "異常系: owner/repo のスラッシュなし",
			subject: "[ownerrepo] Some review (PR #1)",
			wantErr: true,
		},
		{
			name:    "異常系: PR番号が数字でない",
			subject: "[owner/repo] Something (PR #abc)",
			wantErr: true,
		},
		{
			name:    "異常系: repo に / が含まれる（[owner/repo/extra]）",
			subject: "[owner/repo/extra] Something (PR #1)",
			wantErr: true,
		},
		{
			name:    "異常系: subject の前後に余分なテキスト",
			subject: "prefix [owner/repo] Something (PR #1) suffix",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSubject(tt.subject)
			if tt.wantErr {
				if err == nil {
					t.Errorf("エラーを期待したが nil が返った: subject=%q", tt.subject)
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if got.Owner != tt.wantOwner {
				t.Errorf("Owner: got %q, want %q", got.Owner, tt.wantOwner)
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("Repo: got %q, want %q", got.Repo, tt.wantRepo)
			}
			if got.Number != tt.wantNumber {
				t.Errorf("Number: got %d, want %d", got.Number, tt.wantNumber)
			}
		})
	}
}

// CleanBody のテーブル駆動テスト
func TestCleanBody(t *testing.T) {
	// 検証対象: CleanBody  目的: GitHub 通知メールのフッターを除去できること
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "正常系: フッターなし（そのまま返る）",
			body: "レビューコメントの本文です。",
			want: "レビューコメントの本文です。",
		},
		{
			name: "正常系: You are receiving this because... を除去",
			body: "本文テキスト\nYou are receiving this because you were mentioned.\n...",
			want: "本文テキスト",
		},
		{
			name: "正常系: -- \\nYou are receiving... を除去",
			body: "本文テキスト\n-- \nYou are receiving this because you authored the thread.",
			want: "本文テキスト",
		},
		{
			name: "正常系: Unsubscribe from this list を除去",
			body: "コメント内容\nUnsubscribe from this list.",
			want: "コメント内容",
		},
		{
			name: "正常系: 前後の空白をトリム",
			body: "  本文  \nYou are receiving this because of something.",
			want: "本文",
		},
		{
			name: "正常系: 空文字列",
			body: "",
			want: "",
		},
		{
			name: "正常系: CRLF 本文のフッター除去",
			body: "コメント内容\r\nYou are receiving this because you were mentioned.\r\n",
			want: "コメント内容",
		},
		{
			name: "正常系: 複数 marker が存在する場合は最も手前で切る",
			body: "本文\nUnsubscribe from this list\nYou are receiving this because of something.\n",
			want: "本文",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 検証対象: CleanBody  目的: フッター除去後に正しいテキストが返ること
			got := CleanBody(tt.body)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
