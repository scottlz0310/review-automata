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
		nilCmd    bool // true の場合 Commander に nil を渡す
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
			wantCalls: [][2]string{
				{"/repo", "rev-parse --verify pr-42"},
				{"/repo", "fetch origin pull/42/head:pr-42"},
			},
		},
		{
			name:     "STOP: checkout 失敗",
			prNumber: 99,
			// rev-parse=失敗(ブランチなし), fetch=成功, checkout=失敗
			cmdErrors: []error{errors.New("no branch"), nil, errors.New("checkout error")},
			wantErr:   "PR ブランチの checkout 失敗",
			wantCalls: [][2]string{
				{"/repo", "rev-parse --verify pr-99"},
				{"/repo", "fetch origin pull/99/head:pr-99"},
				{"/repo", "checkout pr-99"},
			},
		},
		{
			name:      "STOP: prNumber が 0 以下",
			prNumber:  0,
			cmdErrors: nil,
			wantErr:   "PR番号が不正です",
			wantCalls: nil, // 入力検証でSTOP: コマンド呼び出しなし
		},
		{
			name:     "STOP: 既存ブランチが存在する (ErrBranchExists)",
			prNumber: 42,
			// rev-parse=成功（ブランチ存在）
			cmdErrors: []error{nil},
			wantErr:   git.ErrBranchExists.Error(),
			wantCalls: [][2]string{
				{"/repo", "rev-parse --verify pr-42"},
			},
		},
		{
			name:      "STOP: dir が空文字",
			prNumber:  42,
			cmdErrors: nil,
			wantErr:   "対象ディレクトリが未指定です",
			wantCalls: nil, // 入力検証でSTOP: コマンド呼び出しなし
		},
		{
			name:      "STOP: Commander が未設定 (nil)",
			prNumber:  42,
			nilCmd:    true,
			cmdErrors: nil,
			wantErr:   "Commander が未設定です",
			wantCalls: nil, // 入力検証でSTOP: コマンド呼び出しなし
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := "/repo"
			if tt.wantErr == "対象ディレクトリが未指定です" {
				dir = ""
			}
			mockCmd := &mockCommander{errors: tt.cmdErrors}

			var cmdArg git.Commander
			if !tt.nilCmd {
				cmdArg = mockCmd
			}
			err := git.FetchAndCheckout(dir, tt.prNumber, cmdArg)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("エラーを期待しましたが got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			// 成功系・失敗系の両方で、意図した位置で STOP しているかを検証
			// nilCmd の場合はコマンド呼び出しが発生しないため calls は確認不要
			if !tt.nilCmd {
				if len(mockCmd.calls) != len(tt.wantCalls) {
					t.Errorf("呼び出し回数不一致: got %d, want %d", len(mockCmd.calls), len(tt.wantCalls))
					return
				}
				for i, call := range mockCmd.calls {
					if call != tt.wantCalls[i] {
						t.Errorf("呼び出し[%d] 不一致: got %v, want %v", i, call, tt.wantCalls[i])
					}
				}
			}
		})
	}
}

func TestFetchAndCheckout_ErrBranchExists(t *testing.T) {
	// 検証対象: FetchAndCheckout  目的: 既存ブランチ時に errors.Is で ErrBranchExists を判定できること
	mockCmd := &mockCommander{errors: []error{nil}} // rev-parse 成功 = ブランチ存在
	err := git.FetchAndCheckout("/repo", 42, mockCmd)
	if err == nil {
		t.Fatal("エラーを期待しましたが got nil")
	}
	if !errors.Is(err, git.ErrBranchExists) {
		t.Errorf("errors.Is(err, ErrBranchExists) = false, got %v", err)
	}
}

func TestForceUpdate(t *testing.T) {
	// 検証対象: ForceUpdate  目的: 入力検証・fetch --force + checkout の正常系・失敗系を網羅
	tests := []struct {
		name      string
		dir       string
		prNumber  int
		nilCmd    bool
		cmdErrors []error
		wantErr   string
		wantCalls [][2]string
	}{
		{
			name:      "正常: fetch --force と checkout が成功",
			dir:       "/repo",
			prNumber:  42,
			cmdErrors: []error{nil, nil, nil},
			wantCalls: [][2]string{
				{"/repo", "checkout main"},
				{"/repo", "fetch origin --force pull/42/head:pr-42"},
				{"/repo", "checkout pr-42"},
			},
		},
		{
			name:      "STOP: checkout main 失敗",
			dir:       "/repo",
			prNumber:  42,
			cmdErrors: []error{errors.New("dirty working tree")},
			wantErr:   "main への切り替え失敗",
			wantCalls: [][2]string{
				{"/repo", "checkout main"},
			},
		},
		{
			name:      "STOP: fetch --force 失敗",
			dir:       "/repo",
			prNumber:  42,
			cmdErrors: []error{nil, errors.New("network error")},
			wantErr:   "PR ブランチの強制取得失敗",
			wantCalls: [][2]string{
				{"/repo", "checkout main"},
				{"/repo", "fetch origin --force pull/42/head:pr-42"},
			},
		},
		{
			name:      "STOP: checkout 失敗",
			dir:       "/repo",
			prNumber:  42,
			cmdErrors: []error{nil, nil, errors.New("checkout error")},
			wantErr:   "PR ブランチの checkout 失敗",
			wantCalls: [][2]string{
				{"/repo", "checkout main"},
				{"/repo", "fetch origin --force pull/42/head:pr-42"},
				{"/repo", "checkout pr-42"},
			},
		},
		{
			name:     "STOP: dir が空文字",
			dir:      "",
			prNumber: 42,
			wantErr:  "対象ディレクトリが未指定です",
		},
		{
			name:     "STOP: prNumber が 0 以下",
			dir:      "/repo",
			prNumber: 0,
			wantErr:  "PR番号が不正です",
		},
		{
			name:     "STOP: Commander が nil",
			dir:      "/repo",
			prNumber: 42,
			nilCmd:   true,
			wantErr:  "Commander が未設定です",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCmd := &mockCommander{errors: tt.cmdErrors}
			var cmdArg git.Commander
			if !tt.nilCmd {
				cmdArg = mockCmd
			}
			err := git.ForceUpdate(tt.dir, tt.prNumber, cmdArg)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("エラーを期待しましたが got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			if !tt.nilCmd && tt.wantCalls != nil {
				if len(mockCmd.calls) != len(tt.wantCalls) {
					t.Errorf("呼び出し回数不一致: got %d, want %d", len(mockCmd.calls), len(tt.wantCalls))
					return
				}
				for i, call := range mockCmd.calls {
					if call != tt.wantCalls[i] {
						t.Errorf("呼び出し[%d] 不一致: got %v, want %v", i, call, tt.wantCalls[i])
					}
				}
			}
		})
	}
}
