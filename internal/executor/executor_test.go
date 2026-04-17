package executor

import (
	"errors"
	"strings"
	"testing"
)

// モック実装

type mockProcessManager struct {
	running bool
	runErr  error
	killed  bool
	killErr error
}

func (m *mockProcessManager) IsRunning(_ []string) (bool, error) {
	return m.running, m.runErr
}

func (m *mockProcessManager) Kill(_ []string) error {
	m.killed = true
	return m.killErr
}

type mockCLIRunner struct {
	capturedStdin string
	capturedDir   string
	err           error
}

func (m *mockCLIRunner) RunWithStdin(stdin, dir string) error {
	m.capturedStdin = stdin
	m.capturedDir = dir
	return m.err
}

// TestBuildPrompt はプロンプト生成が仕様通りの形式であることを検証します。
// 検証対象: BuildPrompt  目的: ISSUE#5 プロンプトテンプレート仕様準拠
func TestBuildPrompt(t *testing.T) {
	t.Parallel()

	exc := New(&mockProcessManager{}, &mockCLIRunner{})

	tests := []struct {
		name       string
		owner      string
		repo       string
		prNumber   int
		body       string
		wantSubstr []string
	}{
		{
			name:     "基本ケース",
			owner:    "scottlz0310",
			repo:     "review-automata",
			prNumber: 15,
			body:     "テストレビューコメント",
			wantSubstr: []string{
				"Task: Apply Copilot review suggestions",
				"Repository: scottlz0310/review-automata",
				"PR: #15",
				"Instructions:",
				"## Review comments:",
				"テストレビューコメント",
			},
		},
		{
			name:     "本文空",
			owner:    "owner",
			repo:     "repo",
			prNumber: 1,
			body:     "",
			wantSubstr: []string{
				"Repository: owner/repo",
				"PR: #1",
			},
		},
		{
			name:     "PR 番号が正確に埋め込まれる",
			owner:    "o",
			repo:     "r",
			prNumber: 999,
			body:     "body",
			wantSubstr: []string{
				"PR: #999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := exc.BuildPrompt(tt.owner, tt.repo, tt.prNumber, tt.body)
			for _, want := range tt.wantSubstr {
				if !strings.Contains(got, want) {
					t.Errorf("BuildPrompt() に %q が含まれない\ngot:\n%s", want, got)
				}
			}
		})
	}
}

// TestIsAgentRunning は ProcessManager への委譲と戻り値が正しいことを検証します。
// 検証対象: IsAgentRunning  目的: プロセス管理の抽象化確認
func TestIsAgentRunning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mockRunning bool
		mockErr     error
		wantRunning bool
		wantErr     bool
	}{
		{
			name:        "起動中",
			mockRunning: true,
			wantRunning: true,
		},
		{
			name:        "未起動",
			mockRunning: false,
			wantRunning: false,
		},
		{
			name:    "プロセス一覧取得失敗",
			mockErr: errors.New("tasklist 失敗"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proc := &mockProcessManager{running: tt.mockRunning, runErr: tt.mockErr}
			exc := New(proc, &mockCLIRunner{})
			got, err := exc.IsAgentRunning()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsAgentRunning() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantRunning {
				t.Errorf("IsAgentRunning() = %v, want %v", got, tt.wantRunning)
			}
		})
	}
}

// TestKillAgent は ProcessManager.Kill への委譲を検証します。
// 検証対象: KillAgent  目的: kill 操作の正常系・異常系
func TestKillAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		killErr error
		wantErr bool
	}{
		{name: "正常終了", killErr: nil},
		{name: "kill 失敗", killErr: errors.New("権限なし"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proc := &mockProcessManager{killErr: tt.killErr}
			exc := New(proc, &mockCLIRunner{})
			err := exc.KillAgent()
			if (err != nil) != tt.wantErr {
				t.Errorf("KillAgent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !proc.killed {
				t.Error("KillAgent() が ProcessManager.Kill を呼ばなかった")
			}
		})
	}
}

// TestRun は CLIRunner への委譲と終了コードハンドリングを検証します。
// 検証対象: Run  目的: プロンプト構築 + CLI 委譲の統合確認
func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		owner, repo string
		prNumber    int
		body        string
		repoPath    string
		runErr      error
		wantErr     bool
		wantSubstr  string
		wantDir     string
	}{
		{
			name:       "正常実行: stdin にリポジトリ情報が含まれる",
			owner:      "scottlz0310",
			repo:       "review-automata",
			prNumber:   15,
			body:       "コメント",
			repoPath:   "/home/user/src/review-automata",
			wantSubstr: "scottlz0310/review-automata",
			wantDir:    "/home/user/src/review-automata",
		},
		{
			name:       "正常実行: stdin にレビュー本文が含まれる",
			owner:      "o",
			repo:       "r",
			prNumber:   1,
			body:       "レビュー本文テスト",
			repoPath:   "/tmp/repo",
			wantSubstr: "レビュー本文テスト",
		},
		{
			name:     "CLI 失敗: エラーを返す",
			owner:    "o",
			repo:     "r",
			prNumber: 1,
			body:     "body",
			runErr:   errors.New("claude not found"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runner := &mockCLIRunner{err: tt.runErr}
			exc := New(&mockProcessManager{}, runner)
			err := exc.Run(tt.owner, tt.repo, tt.prNumber, tt.body, tt.repoPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantSubstr != "" && !strings.Contains(runner.capturedStdin, tt.wantSubstr) {
				t.Errorf("Run() に渡された stdin に %q が含まれない\ngot:\n%s", tt.wantSubstr, runner.capturedStdin)
			}
			if tt.wantDir != "" && runner.capturedDir != tt.wantDir {
				t.Errorf("Run() に渡された dir = %q, want %q", runner.capturedDir, tt.wantDir)
			}
		})
	}
}
