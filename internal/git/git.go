// Package git は、PR ブランチの取得およびチェックアウト操作を扱います。
package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrBranchExists は対象の PR ブランチが既にローカルに存在することを示します。
// 呼び出し元は errors.Is(err, ErrBranchExists) で判定できます。
var ErrBranchExists = errors.New("ブランチが既に存在します")

// Commander は git コマンドの実行を抽象化します。
type Commander interface {
	// Run は dir ディレクトリで git コマンドを実行し、stdout を返します。
	Run(dir string, args ...string) (string, error)
}

// ExecCommander は os/exec を使った Commander の実装です。
type ExecCommander struct{}

// Run は指定ディレクトリで git コマンドを実行します。
func (ExecCommander) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		if output == "" {
			return "", fmt.Errorf("git コマンド実行失敗 (dir=%s, args=%s): %w", dir, strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git コマンド実行失敗 (dir=%s, args=%s): %w: %s", dir, strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(out)), nil
}

// FetchAndCheckout は PR ブランチを取得してチェックアウトします。
// 実行するコマンド:
//
//	git fetch origin pull/{prNumber}/head:pr-{prNumber}
//	git checkout pr-{prNumber}
//
// 入力値不正・既存ブランチ・fetch/checkout の失敗は STOP 条件として error を返します。
func FetchAndCheckout(dir string, prNumber int, cmd Commander) error {
	if dir == "" {
		return fmt.Errorf("対象ディレクトリが未指定です")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR番号が不正です: %d", prNumber)
	}
	if cmd == nil {
		return fmt.Errorf("Commander が未設定です")
	}

	// 既存ブランチの有無を事前確認: 存在する場合は ErrBranchExists を返す
	branch := fmt.Sprintf("pr-%d", prNumber)
	if _, err := cmd.Run(dir, "rev-parse", "--verify", branch); err == nil {
		return fmt.Errorf("ブランチ %q %w", branch, ErrBranchExists)
	}

	refSpec := fmt.Sprintf("pull/%d/head:pr-%d", prNumber, prNumber)
	if _, err := cmd.Run(dir, "fetch", "origin", refSpec); err != nil {
		return fmt.Errorf("PR ブランチの取得失敗 (PR #%d): %w", prNumber, err)
	}

	if _, err := cmd.Run(dir, "checkout", branch); err != nil {
		return fmt.Errorf("PR ブランチの checkout 失敗 (%s): %w", branch, err)
	}

	return nil
}

// ForceUpdate は既存の PR ブランチを強制的に最新化してチェックアウトします。
// fetch --force でローカルブランチを上書きした後 checkout します。
// エージェント起動確認済み・または未起動の場合に呼び出してください。
func ForceUpdate(dir string, prNumber int, cmd Commander) error {
	if dir == "" {
		return fmt.Errorf("対象ディレクトリが未指定です")
	}
	if prNumber <= 0 {
		return fmt.Errorf("PR番号が不正です: %d", prNumber)
	}
	if cmd == nil {
		return fmt.Errorf("Commander が未設定です")
	}

	branch := fmt.Sprintf("pr-%d", prNumber)
	refSpec := fmt.Sprintf("pull/%d/head:%s", prNumber, branch)
	if _, err := cmd.Run(dir, "fetch", "origin", "--force", refSpec); err != nil {
		return fmt.Errorf("PR ブランチの強制取得失敗 (PR #%d): %w", prNumber, err)
	}

	if _, err := cmd.Run(dir, "checkout", branch); err != nil {
		return fmt.Errorf("PR ブランチの checkout 失敗 (%s): %w", branch, err)
	}

	return nil
}
