// Package executor はプロンプトを構築し、標準入力経由で Claude CLI に処理を委譲します。
package executor

import (
	"fmt"
	"os/exec"
	"strings"
)

// agentProcessNames は起動確認対象のエージェントプロセス名一覧です。
// 環境変数等による設定切り替えは v0.5.0 以降で対応します。
var agentProcessNames = []string{"claude"}

// IsAgentRunning は設定済みエージェント CLI のプロセスが起動中かどうかを返します。
// プロセス一覧の取得に失敗した場合はエラーを返します。
// 呼び出し元は error != nil の場合も安全側（確認プロンプトあり）で処理してください。
func IsAgentRunning() (bool, error) {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false, fmt.Errorf("プロセス一覧の取得失敗: %w", err)
	}
	lower := strings.ToLower(string(out))
	for _, name := range agentProcessNames {
		if strings.Contains(lower, strings.ToLower(name)+".exe") {
			return true, nil
		}
	}
	return false, nil
}

// KillAgent は起動中のエージェント CLI プロセスを強制終了します。
// 対象プロセスが見つからない場合（既に終了済み等）はエラーを返しません。
// taskkill の実行自体に失敗した場合はエラーを返します。
func KillAgent() error {
	for _, name := range agentProcessNames {
		out, err := exec.Command("taskkill", "/F", "/IM", name+".exe").CombinedOutput()
		if err != nil {
			msg := strings.ToLower(string(out))
			// プロセスが見つからない場合（既に終了済み）は正常扱い
			if strings.Contains(msg, "not found") || strings.Contains(msg, "見つかりません") {
				continue
			}
			return fmt.Errorf("エージェント CLI の強制終了失敗 (%s): %w: %s", name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}
