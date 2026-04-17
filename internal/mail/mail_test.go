package mail

import (
	"context"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
)

// 検証対象: Config.validate  目的: 必須フィールド欠落および補完動作を検証する
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		wantErr     bool
		errContains string
		wantMailbox string // validate 後の Mailbox 期待値（空の場合はチェックしない）
	}{
		{
			name:        "正常: 全フィールド設定済み",
			cfg:         Config{Addr: "imap.example.com:993", Username: "user@example.com", Password: "pass", Mailbox: "INBOX"},
			wantErr:     false,
			wantMailbox: "INBOX",
		},
		{
			name:        "正常: Mailbox 未設定は INBOX に補完される",
			cfg:         Config{Addr: "imap.example.com:993", Username: "user@example.com", Password: "pass"},
			wantErr:     false,
			wantMailbox: "INBOX",
		},
		{
			name:        "異常: Addr 未設定",
			cfg:         Config{Username: "user@example.com", Password: "pass"},
			wantErr:     true,
			errContains: "IMAP アドレスが未設定です",
		},
		{
			name:        "異常: Username 未設定",
			cfg:         Config{Addr: "imap.example.com:993", Password: "pass"},
			wantErr:     true,
			errContains: "ユーザー名が未設定です",
		},
		{
			name:        "異常: Password 未設定",
			cfg:         Config{Addr: "imap.example.com:993", Username: "user@example.com"},
			wantErr:     true,
			errContains: "パスワードが未設定です",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("エラーが期待されるが nil が返された")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}
			if tt.wantMailbox != "" && tt.cfg.Mailbox != tt.wantMailbox {
				t.Errorf("Mailbox 補完失敗: got %q, want %q", tt.cfg.Mailbox, tt.wantMailbox)
			}
		})
	}
}

// 検証対象: Watcher.Watch  目的: 設定エラーは接続前に即座に error を返す
func TestWatchInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		cfg         Config
		errContains string
	}{
		{
			name:        "全フィールド空",
			cfg:         Config{},
			errContains: "IMAP 設定エラー",
		},
		{
			name:        "Username・Password 未設定",
			cfg:         Config{Addr: "imap.gmail.com:993"},
			errContains: "IMAP 設定エラー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := New(tt.cfg)
			err := w.Watch(context.Background(), func(_, _ string) error { return nil })
			if err == nil {
				t.Fatal("設定エラーで Watch が nil を返した")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.errContains)
			}
		})
	}
}

// 検証対象: extractTextBody  目的: text/plain 部分のみ抽出される
func TestExtractTextBody(t *testing.T) {
	tests := []struct {
		name     string
		raw      string // RFC 2822 形式のメッセージ
		wantBody string
	}{
		{
			name: "text/plain 単一パート",
			raw: "MIME-Version: 1.0\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"\r\n" +
				"Hello, World!\r\n",
			wantBody: "Hello, World!",
		},
		{
			name: "multipart/alternative: text/plain を抽出",
			raw: "MIME-Version: 1.0\r\n" +
				"Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n" +
				"\r\n" +
				"--boundary123\r\n" +
				"Content-Type: text/plain; charset=utf-8\r\n" +
				"\r\n" +
				"Plain text content\r\n" +
				"--boundary123\r\n" +
				"Content-Type: text/html; charset=utf-8\r\n" +
				"\r\n" +
				"<p>HTML content</p>\r\n" +
				"--boundary123--\r\n",
			wantBody: "Plain text content",
		},
		{
			name: "text/html のみ: 空文字を返す",
			raw: "MIME-Version: 1.0\r\n" +
				"Content-Type: text/html; charset=utf-8\r\n" +
				"\r\n" +
				"<p>HTML only</p>\r\n",
			wantBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// go-imap の Message 構造体にボディをセット
			section := &imap.BodySectionName{}
			msg := &imap.Message{
				Body: map[*imap.BodySectionName]imap.Literal{
					section: strings.NewReader(tt.raw),
				},
			}

			got := extractTextBody(msg, section)
			got = strings.TrimSpace(got)
			if got != tt.wantBody {
				t.Errorf("body 不一致:\n  got:  %q\n  want: %q", got, tt.wantBody)
			}
		})
	}
}
