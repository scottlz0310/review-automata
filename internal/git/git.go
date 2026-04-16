// Package git は、PR ブランチの取得およびチェックアウト操作を扱います。
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

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
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// FetchAndCheckout は PR ブランチを取得してチェックアウトします。
// 実行するコマンド:
//
//	git fetch origin pull/{prNumber}/head:pr-{prNumber}
//	git checkout pr-{prNumber}
//
// fetch または checkout が失敗した場合は STOP 条件として error を返します。
func FetchAndCheckout(dir string, prNumber int, cmd Commander) error {
	refSpec := fmt.Sprintf("pull/%d/head:pr-%d", prNumber, prNumber)
	if _, err := cmd.Run(dir, "fetch", "origin", refSpec); err != nil {
		return fmt.Errorf("PR ブランチの取得失敗 (PR #%d): %w", prNumber, err)
	}

	branch := fmt.Sprintf("pr-%d", prNumber)
	if _, err := cmd.Run(dir, "checkout", branch); err != nil {
		return fmt.Errorf("PR ブランチの checkout 失敗 (%s): %w", branch, err)
	}

	return nil
}
