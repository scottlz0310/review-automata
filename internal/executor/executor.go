// Package executor はプロンプトを構築し、標準入力経由で Claude CLI に処理を委譲します。
package executor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ProcessManager はプロセス管理操作の抽象インターフェースです。
// 外部依存（tasklist / taskkill）をモック差し替えするために使用します。
type ProcessManager interface {
	IsRunning(names []string) (bool, error)
	Kill(names []string) error
}

// CLIRunner は外部 CLI 実行の抽象インターフェースです。
// stdin と作業ディレクトリを受け取り claude CLI に委譲します。テストでのモック差し替えに使用します。
type CLIRunner interface {
	RunWithStdin(stdin, dir string) error
}

// ExecProcessManager は実際の tasklist / taskkill コマンドを使う ProcessManager 実装です。
type ExecProcessManager struct{}

// IsRunning は names のうちいずれかのプロセスが起動中かどうかを返します。
// tasklist の取得自体に失敗した場合はエラーを返します。
func (ExecProcessManager) IsRunning(names []string) (bool, error) {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false, fmt.Errorf("プロセス一覧の取得失敗: %w", err)
	}
	lower := strings.ToLower(string(out))
	for _, name := range names {
		if strings.Contains(lower, strings.ToLower(name)+".exe") {
			return true, nil
		}
	}
	return false, nil
}

// Kill は names のプロセスを強制終了します。
// プロセスが見つからない場合は正常扱いです。taskkill の実行失敗はエラーを返します。
func (ExecProcessManager) Kill(names []string) error {
	for _, name := range names {
		out, err := exec.Command("taskkill", "/F", "/IM", name+".exe").CombinedOutput()
		if err != nil {
			msg := strings.ToLower(string(out))
			if strings.Contains(msg, "not found") || strings.Contains(msg, "見つかりません") {
				continue
			}
			return fmt.Errorf("エージェント CLI の強制終了失敗 (%s): %w: %s", name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// ExecCLIRunner は実際の claude コマンドを実行する CLIRunner 実装です。
type ExecCLIRunner struct{}

// RunWithStdin は stdin をパイプして claude CLI を起動します。
// --print フラグにより非対話モードで実行し、stdin のプロンプトを処理させます。
// dir が空でない場合は cmd.Dir に設定し、対象リポジトリのディレクトリで実行します。
// 非ゼロ終了コードはエラーとして返します。
func (ExecCLIRunner) RunWithStdin(stdin, dir string) error {
	cmd := exec.Command("claude", "--print")
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("claude CLI が終了コード %d で終了しました: %w", exitErr.ExitCode(), err)
		}
		return fmt.Errorf("claude CLI の起動失敗: %w", err)
	}
	return nil
}

// defaultAgentNames はデフォルトのエージェントプロセス名一覧です。
var defaultAgentNames = []string{"claude"}

// Executor はプロンプト構築と Claude CLI 委譲を担当します。
type Executor struct {
	proc   ProcessManager
	runner CLIRunner
	names  []string
}

// New は Executor を構築します。
// proc にプロセス管理実装、runner に CLI 実行実装を注入します。
func New(proc ProcessManager, runner CLIRunner) *Executor {
	return &Executor{
		proc:   proc,
		runner: runner,
		names:  defaultAgentNames,
	}
}

// IsAgentRunning はエージェント CLI が起動中かどうかを返します。
// プロセス一覧の取得に失敗した場合はエラーを返します。
// 呼び出し元は error != nil の場合も安全側（確認プロンプトあり）で処理してください。
func (e *Executor) IsAgentRunning() (bool, error) {
	return e.proc.IsRunning(e.names)
}

// KillAgent は起動中のエージェント CLI を強制終了します。
// プロセスが見つからない場合（既に終了済み等）はエラーを返しません。
func (e *Executor) KillAgent() error {
	return e.proc.Kill(e.names)
}

// BuildPrompt はメタ情報とレビュー本文からプロンプト文字列を構築します。
// テンプレート仕様は Mcp-Docker#38 に準拠します。
func (e *Executor) BuildPrompt(owner, repo string, prNumber int, body string) string {
	return fmt.Sprintf(
		"Task: Apply Copilot review suggestions\n\nRepository: %s/%s\nPR: #%d\n\nInstructions:\n- 指摘内容を修正する\n- 既存の挙動を壊さない\n- 最小限の変更に留める\n\n## Review comments:\n\n%s",
		owner, repo, prNumber, body,
	)
}

// Run はプロンプトを構築して claude CLI に STDIN 経由で委譲します。
// repoPath は claude CLI の作業ディレクトリとして設定します。
func (e *Executor) Run(owner, repo string, prNumber int, body, repoPath string) error {
	prompt := e.BuildPrompt(owner, repo, prNumber, body)
	if err := e.runner.RunWithStdin(prompt, repoPath); err != nil {
		return fmt.Errorf("PR #%d (%s/%s) の claude CLI 実行失敗: %w", prNumber, owner, repo, err)
	}
	return nil
}
