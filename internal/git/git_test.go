package git_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/scottlz0310/review-automata/internal/git"
)

// mockCommander は Commander のモック実装です。
type mockCommander struct {
	callIdx int
	errors  []error
	calls   [][2]string // [dir, "arg1 arg2 ..."]
}

func (m *mockCommander) Run(dir string, args ...string) (string, error) {
	m.calls = append(m.calls, [2]string{dir, strings.Join(args, " ")})
	idx := m.callIdx
	m.callIdx++
	if idx < len(m.errors) {
		return "", m.errors[idx]
	}
	return "", nil
}

func TestFetchAndCheckout(t *testing.T) {
	// 検証対象: FetchAndCheckout  目的: 入力検証・既存ブランチSTOP・fetch/checkoutの正常系・STOP条件を網羅
	tests := []struct {
		name      string
		prNumber  int
		cmdErrors []error
		wantErr   string      // 空文字なら正常系
		wantCalls [][2]string // 期待するコマンド呼び出し [dir, args]
	}{
		{
			name:     "正常: fetch と checkout が成功",
			prNumber: 42,
			// rev-parse=失敗(ブランチなし), fetch=成功, checkout=成功
			cmdErrors: []error{errors.New("no branch"), nil, nil},
			wantCalls: [][2]string{
				{"/repo", "rev-parse --verify pr-42"},
				{"/repo", "fetch origin pull/42/head:pr-42"},
				{"/repo", "checkout pr-42"},
			},
		},
		{
			name:     "STOP: fetch 失敗",
			prNumber: 42,
			// rev-parse=失敗(ブランチなし), fetch=失敗
			cmdErrors: []error{errors.New("no branch"), errors.New("network error")},
			wantErr:   "PR ブランチの取得失敗",
		},
		{
			name:     "STOP: checkout 失敗",
			prNumber: 99,
			// rev-parse=失敗(ブランチなし), fetch=成功, checkout=失敗
			cmdErrors: []error{errors.New("no branch"), nil, errors.New("checkout error")},
			wantErr:   "PR ブランチの checkout 失敗",
		},
		{
			name:      "STOP: prNumber が 0 以下",
			prNumber:  0,
			cmdErrors: nil,
			wantErr:   "PR番号が不正です",
		},
		{
			name:     "STOP: 既存ブランチが存在する",
			prNumber: 42,
			// rev-parse=成功（ブランチ存在）
			cmdErrors: []error{nil},
			wantErr:   "が既に存在します",
		},
		{
			name:      "STOP: dir が空文字",
			prNumber:  42,
			cmdErrors: nil,
			wantErr:   "対象ディレクトリが未指定です",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := "/repo"
			if tt.wantErr == "対象ディレクトリが未指定です" {
				dir = ""
			}
			cmd := &mockCommander{errors: tt.cmdErrors}
			err := git.FetchAndCheckout(dir, tt.prNumber, cmd)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("エラーを期待しましたが got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			// 正常系: コマンド呼び出し内容を検証
			if len(cmd.calls) != len(tt.wantCalls) {
				t.Errorf("呼び出し回数不一致: got %d, want %d", len(cmd.calls), len(tt.wantCalls))
				return
			}
			for i, call := range cmd.calls {
				if call != tt.wantCalls[i] {
					t.Errorf("呼び出し[%d] 不一致: got %v, want %v", i, call, tt.wantCalls[i])
				}
			}
		})
	}
}
