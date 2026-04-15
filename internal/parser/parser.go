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
// - 返信メールの "Re: " プレフィックスも許容します。
// - owner・repo ともに "/" を含まない文字列のみ受理し、曖昧な subject は STOP します。
// - subject 全体に一致する場合のみ受理します（先頭 ^ / 末尾 $ アンカー）。
var subjectPattern = regexp.MustCompile(`^(?:Re:\s*)?\[([^/\]]+)/([^/\]]+)\].*\(PR #(\d+)\)$`)

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

// cutMarkers は GitHub 通知メールのフッター開始を示す文字列のリストです。
var cutMarkers = []string{
	"-- \nYou are receiving this because",
	"---\nYou are receiving this because",
	"You are receiving this because",
	"Unsubscribe from this list",
	"Manage your GitHub notification settings",
}

// CleanBody は GitHub 通知メールの本文から不要なフッターを除去します。
// CRLF は LF に正規化してから処理します。構造化は行わず、テキストのみを返します。
func CleanBody(body string) string {
	// CRLF → LF に正規化
	body = strings.ReplaceAll(body, "\r\n", "\n")

	// 全 marker の中で最も手前に現れる位置で切る
	cutIndex := len(body)
	found := false
	for _, marker := range cutMarkers {
		if idx := strings.Index(body, marker); idx != -1 && idx < cutIndex {
			cutIndex = idx
			found = true
		}
	}

	if found {
		return strings.TrimSpace(body[:cutIndex])
	}
	return strings.TrimSpace(body)
}
