// Package parser は Copilot のレビュー通知メールから PR のメタデータを抽出します。
package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PRMetadata はメールの subject から抽出した PR 情報を保持します。
type PRMetadata struct {
	Owner  string
	Repo   string
	Number int
}

// subjectPattern は GitHub 通知メールの subject から owner / repo / PR番号を抽出します。
// 対象フォーマット: [owner/repo] ... (PR #123)
var subjectPattern = regexp.MustCompile(`\[([^/\]]+)/([^\]]+)\].*\(PR #(\d+)\)`)

// ParseSubject は subject 文字列を解析し、PRMetadata を返します。
// パースに失敗した場合は STOP 条件として error を返します。
func ParseSubject(subject string) (*PRMetadata, error) {
	m := subjectPattern.FindStringSubmatch(subject)
	if m == nil {
		return nil, fmt.Errorf("subject パース失敗: 期待するフォーマットに一致しません: %q", subject)
	}
	number, err := strconv.Atoi(m[3])
	if err != nil {
		// 正規表現で \d+ を保証しているため通常到達しない
		return nil, fmt.Errorf("PR番号の変換失敗: %w", err)
	}
	return &PRMetadata{
		Owner:  m[1],
		Repo:   m[2],
		Number: number,
	}, nil
}

// CleanBody は GitHub 通知メールの本文から不要なフッター／ヘッダーを除去します。
// 構造化は行わず、テキストのみを返します。
func CleanBody(body string) string {
	// GitHub 通知メールに共通するフッター区切り文字列
	cutMarkers := []string{
		"-- \nYou are receiving this because",
		"---\nYou are receiving this because",
		"You are receiving this because",
		"Unsubscribe from this list",
		"Manage your GitHub notification settings",
	}
	result := body
	for _, marker := range cutMarkers {
		if idx := strings.Index(result, marker); idx != -1 {
			result = result[:idx]
		}
	}
	return strings.TrimSpace(result)
}
