// Package resolver は、名前と origin URL をもとに ~/src 配下の Git リポジトリを特定します。
package resolver

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRunner は git remote の URL 取得操作を抽象化します。
type GitRunner interface {
	GetOriginURL(dir string) (string, error)
}

// ExecGitRunner は os/exec を使った GitRunner の実装です。
type ExecGitRunner struct{}

// GetOriginURL は git remote get-url origin でリモート URL を取得します。
func (ExecGitRunner) GetOriginURL(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").CombinedOutput()
	if err != nil {
		formatted := formatCommandOutput(out)
		if formatted != "" {
			return "", fmt.Errorf("origin URL の取得失敗 (%s): %s: %w", dir, formatted, err)
		}
		return "", fmt.Errorf("origin URL の取得失敗 (%s): %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// formatCommandOutput はコマンド出力をエラーメッセージ向けに整形します。
func formatCommandOutput(out []byte) string {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return ""
	}
	s = strings.Join(strings.Fields(s), " ")
	const maxLen = 200
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// Resolver は ~/src 配下のリポジトリを特定します。
type Resolver struct {
	BaseDir   string
	GitRunner GitRunner
}

// New は Resolver を初期化します。BaseDir のデフォルトは ~/src です。
// runner が nil の場合は ExecGitRunner をデフォルトとして使用します。
func New(runner GitRunner) (*Resolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("ホームディレクトリの取得失敗: %w", err)
	}
	if runner == nil {
		runner = ExecGitRunner{}
	}
	return &Resolver{
		BaseDir:   filepath.Join(home, "src"),
		GitRunner: runner,
	}, nil
}

// Resolve は owner/repo に一致するローカルリポジトリのパスを返します。
// 以下の場合は STOP 条件として error を返します:
//   - GitRunner 未設定
//   - BaseDir 不在・探索失敗
//   - 候補 0 件
//   - 候補複数かつ一部の origin URL 取得失敗（誤ったリポジトリ選択を防ぐため中止）
//   - 全候補で origin URL 取得失敗
//   - origin 不一致
//   - 一致リポジトリが複数件
func (r *Resolver) Resolve(owner, repo string) (string, error) {
	if r.GitRunner == nil {
		return "", fmt.Errorf("GitRunner が未設定です: Resolver を正しく初期化してください")
	}
	candidates, err := r.findCandidates(repo)
	if err != nil {
		return "", fmt.Errorf("リポジトリ検索失敗: %w", err)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("リポジトリ未検出: %q が %q 配下に見つかりません", repo, r.BaseDir)
	}

	var matched []string
	var originErrs []string
	originCheckedCount := 0
	for _, dir := range candidates {
		url, err := r.GitRunner.GetOriginURL(dir)
		if err != nil {
			originErrs = append(originErrs, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		originCheckedCount++
		if originMatches(url, owner, repo) {
			matched = append(matched, dir)
		}
	}

	if originCheckedCount == 0 {
		return "", fmt.Errorf(
			"origin 取得失敗: %s/%s の候補 %d 件すべてで origin URL を取得できませんでした: %s",
			owner, repo, len(candidates), strings.Join(originErrs, "; "),
		)
	}

	// 候補が複数あり一部で origin 取得失敗した場合、未確認候補が残るため安全のため STOP する
	if len(candidates) > 1 && len(originErrs) > 0 {
		return "", fmt.Errorf(
			"origin 取得失敗: %s/%s は候補が %d 件あり、一部候補の origin URL を取得できないため安全のため特定を中止しました: %s",
			owner, repo, len(candidates), strings.Join(originErrs, "; "),
		)
	}

	switch len(matched) {
	case 0:
		return "", fmt.Errorf("origin 不一致: %s/%s に一致するリポジトリが見つかりません", owner, repo)
	case 1:
		return matched[0], nil
	default:
		return "", fmt.Errorf("リポジトリ複数検出: %s/%s に一致するリポジトリが %d 件あります: %v", owner, repo, len(matched), matched)
	}
}

// findCandidates は BaseDir 配下から repo と同名のディレクトリを探します。
func (r *Resolver) findCandidates(repo string) ([]string, error) {
	var candidates []string
	err := filepath.WalkDir(r.BaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == r.BaseDir {
				return err // ベースディレクトリ自体のエラーは伝播する
			}
			return filepath.SkipDir // サブディレクトリのアクセスエラーはスキップ
		}
		if d.IsDir() && d.Name() == repo && path != r.BaseDir {
			candidates = append(candidates, path)
			return filepath.SkipDir // サブディレクトリは探索不要
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return candidates, nil
}

// originMatches は git remote URL が owner/repo に一致するか確認します。
// GitHub の HTTPS 形式 (https://github.com/owner/repo[.git]) と
// SSH 形式 (git@github.com:owner/repo[.git]) のみ受け付けます。
func originMatches(url, owner, repo string) bool {
	url = strings.TrimSpace(url)

	const (
		githubHTTPSPrefix = "https://github.com/"
		githubSSHPrefix   = "git@github.com:"
	)

	var repoPath string
	switch {
	case strings.HasPrefix(url, githubHTTPSPrefix):
		repoPath = strings.TrimPrefix(url, githubHTTPSPrefix)
	case strings.HasPrefix(url, githubSSHPrefix):
		repoPath = strings.TrimPrefix(url, githubSSHPrefix)
	default:
		return false
	}

	repoPath = strings.TrimSuffix(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	repoPath = strings.TrimSuffix(repoPath, "/")

	return repoPath == owner+"/"+repo
}
