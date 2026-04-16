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
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("origin URL の取得失敗 (%s): %w", dir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Resolver は ~/src 配下のリポジトリを特定します。
type Resolver struct {
	BaseDir   string
	GitRunner GitRunner
}

// New は Resolver を初期化します。BaseDir のデフォルトは ~/src です。
func New(runner GitRunner) (*Resolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("ホームディレクトリの取得失敗: %w", err)
	}
	return &Resolver{
		BaseDir:   filepath.Join(home, "src"),
		GitRunner: runner,
	}, nil
}

// Resolve は owner/repo に一致するローカルリポジトリのパスを返します。
// 候補が 0件・複数件・origin 不一致の場合は STOP 条件として error を返します。
func (r *Resolver) Resolve(owner, repo string) (string, error) {
	candidates, err := r.findCandidates(repo)
	if err != nil {
		return "", fmt.Errorf("リポジトリ検索失敗: %w", err)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("リポジトリ未検出: %q が %q 配下に見つかりません", repo, r.BaseDir)
	}

	var matched []string
	for _, dir := range candidates {
		url, err := r.GitRunner.GetOriginURL(dir)
		if err != nil {
			continue // origin が取得できないディレクトリはスキップ
		}
		if originMatches(url, owner, repo) {
			matched = append(matched, dir)
		}
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
// HTTPS 形式 (https://github.com/owner/repo[.git]) と
// SSH 形式 (git@github.com:owner/repo[.git]) の両方に対応します。
func originMatches(url, owner, repo string) bool {
	url = strings.TrimSpace(url)
	url = strings.TrimSuffix(url, ".git")
	return strings.HasSuffix(url, owner+"/"+repo)
}
