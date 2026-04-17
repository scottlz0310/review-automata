// Package executor はプロンプトを構築し、標準入力経由で Claude CLI に処理を委譲します。
package executor

import (
	"os/exec"
	"strings"
)

// agentProcessNames は起動確認対象のエージェントプロセス名一覧です。
// 環境変数等による設定切り替えは v0.5.0 以降で対応します。
var agentProcessNames = []string{"claude"}

// IsAgentRunning は設定済みエージェント CLI のプロセスが起動中かどうかを返します。
// プロセス一覧の取得に失敗した場合は false を返します（STOP 優先）。
func IsAgentRunning() bool {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(out))
	for _, name := range agentProcessNames {
		if strings.Contains(lower, strings.ToLower(name)+".exe") {
			return true
		}
	}
	return false
}

// KillAgent は起動中のエージェント CLI プロセスを強制終了します。
// 対象プロセスが存在しない場合はエラーを返しません。
func KillAgent() error {
	for _, name := range agentProcessNames {
		// エラーは無視（プロセスが既に終了している場合もある）
		_ = exec.Command("taskkill", "/F", "/IM", name+".exe").Run()
	}
	return nil
}
