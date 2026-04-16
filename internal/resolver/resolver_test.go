package resolver_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottlz0310/review-automata/internal/resolver"
)

// mockGitRunner は GitRunner のモック実装です。
type mockGitRunner struct {
	urls map[string]string // dir パス -> origin URL
}

func (m *mockGitRunner) GetOriginURL(dir string) (string, error) {
	if url, ok := m.urls[dir]; ok {
		return url, nil
	}
	return "", fmt.Errorf("mock: %s に origin が設定されていません", dir)
}

// makeDir はテスト用のディレクトリ構造を作成し、フルパスを返します。
func makeDir(t *testing.T, base string, parts ...string) string {
	t.Helper()
	full := filepath.Join(append([]string{base}, parts...)...)
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("ディレクトリ作成失敗: %v", err)
	}
	return full
}

func TestResolve(t *testing.T) {
	// 検証対象: Resolver.Resolve  目的: 正常系・STOP条件（0件/複数件/origin不一致/origin取得全失敗/BaseDir不在/GitRunner未設定）を網羅
	tests := []struct {
		name            string
		dirs            [][]string // baseDir 配下に作るサブパス（nil = 作らない）
		nonexistentBase bool       // true の場合 BaseDir を存在しないパスに設定
		owner           string
		repo            string
		setup           func(baseDir string) resolver.GitRunner
		wantErr         string // 空文字なら正常系、含まれることを確認する文字列
	}{
		{
			name:  "正常: 1件一致 (HTTPS URL)",
			dirs:  [][]string{{"owner1", "myrepo"}},
			owner: "owner1",
			repo:  "myrepo",
			setup: func(baseDir string) resolver.GitRunner {
				return &mockGitRunner{urls: map[string]string{
					filepath.Join(baseDir, "owner1", "myrepo"): "https://github.com/owner1/myrepo.git",
				}}
			},
		},
		{
			name:  "正常: 1件一致 (SSH URL)",
			dirs:  [][]string{{"owner2", "myrepo"}},
			owner: "owner2",
			repo:  "myrepo",
			setup: func(baseDir string) resolver.GitRunner {
				return &mockGitRunner{urls: map[string]string{
					filepath.Join(baseDir, "owner2", "myrepo"): "git@github.com:owner2/myrepo",
				}}
			},
		},
		{
			name:    "STOP: リポジトリ未検出 (0件)",
			dirs:    [][]string{{"other", "otherrepo"}},
			owner:   "owner1",
			repo:    "myrepo",
			setup:   func(baseDir string) resolver.GitRunner { return &mockGitRunner{} },
			wantErr: "リポジトリ未検出",
		},
		{
			name:  "STOP: origin 不一致",
			dirs:  [][]string{{"owner1", "myrepo"}},
			owner: "wrong-owner",
			repo:  "myrepo",
			setup: func(baseDir string) resolver.GitRunner {
				return &mockGitRunner{urls: map[string]string{
					filepath.Join(baseDir, "owner1", "myrepo"): "https://github.com/owner1/myrepo.git",
				}}
			},
			wantErr: "origin 不一致",
		},
		{
			name:  "STOP: 複数件検出",
			dirs:  [][]string{{"owner1", "myrepo"}, {"owner2", "myrepo"}},
			owner: "owner1",
			repo:  "myrepo",
			setup: func(baseDir string) resolver.GitRunner {
				return &mockGitRunner{urls: map[string]string{
					filepath.Join(baseDir, "owner1", "myrepo"): "https://github.com/owner1/myrepo.git",
					// owner2 配下にも同じ origin を持つ同名リポジトリが存在する
					filepath.Join(baseDir, "owner2", "myrepo"): "https://github.com/owner1/myrepo.git",
				}}
			},
			wantErr: "リポジトリ複数検出",
		},
		{
			name:            "STOP: BaseDir が存在しない",
			nonexistentBase: true,
			owner:           "owner1",
			repo:            "myrepo",
			setup:           func(baseDir string) resolver.GitRunner { return &mockGitRunner{} },
			wantErr:         "リポジトリ検索失敗",
		},
		{
			name:  "STOP: GitRunner が nil",
			dirs:  [][]string{{"owner1", "myrepo"}},
			owner: "owner1",
			repo:  "myrepo",
			setup: func(baseDir string) resolver.GitRunner { return nil },
			wantErr: "GitRunner が未設定",
		},
		{
			name:  "STOP: 全候補で origin 取得失敗",
			dirs:  [][]string{{"owner1", "myrepo"}},
			owner: "owner1",
			repo:  "myrepo",
			// mockGitRunner の urls に該当エントリがないため GetOriginURL はエラーを返す
			setup:   func(baseDir string) resolver.GitRunner { return &mockGitRunner{} },
			wantErr: "origin 取得失敗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()

			var resolverBaseDir string
			if tt.nonexistentBase {
				resolverBaseDir = filepath.Join(baseDir, "nonexistent")
			} else {
				resolverBaseDir = baseDir
				for _, parts := range tt.dirs {
					makeDir(t, baseDir, parts...)
				}
			}

			runner := tt.setup(baseDir)
			r := &resolver.Resolver{
				BaseDir:   resolverBaseDir,
				GitRunner: runner,
			}

			got, err := r.Resolve(tt.owner, tt.repo)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("エラーを期待しましたが got nil, path=%s", got)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("エラーメッセージ不一致: got %q, want contains %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("予期しないエラー: %v", err)
				return
			}

			// 正常系: 返却パスが作成した唯一のディレクトリに一致することを確認
			expected := makeDir(t, baseDir, tt.dirs[0]...)
			if got != expected {
				t.Errorf("パス不一致: got %q, want %q", got, expected)
			}
		})
	}
}

func TestOriginMatchesVariants(t *testing.T) {
	// 検証対象: originMatches（非公開関数のため Resolve 経由で間接的に検証）
	// 目的: HTTPS/SSH/末尾.git有無/非GitHubホストの各 URL 形式で正しく一致するか確認
	tests := []struct {
		name   string
		url    string
		owner  string
		repo   string
		wantOK bool
	}{
		{"HTTPS .git あり", "https://github.com/owner1/myrepo.git", "owner1", "myrepo", true},
		{"HTTPS .git なし", "https://github.com/owner1/myrepo", "owner1", "myrepo", true},
		{"SSH .git あり", "git@github.com:owner1/myrepo.git", "owner1", "myrepo", true},
		{"SSH .git なし", "git@github.com:owner1/myrepo", "owner1", "myrepo", true},
		{"owner 不一致", "https://github.com/other/myrepo.git", "owner1", "myrepo", false},
		{"repo 不一致", "https://github.com/owner1/other.git", "owner1", "myrepo", false},
		{"GitLab URL は一致しない", "https://gitlab.com/owner1/myrepo.git", "owner1", "myrepo", false},
		{"不明ホスト URL は一致しない", "https://example.com/owner1/myrepo", "owner1", "myrepo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			dir := makeDir(t, baseDir, tt.owner, tt.repo)

			r := &resolver.Resolver{
				BaseDir: baseDir,
				GitRunner: &mockGitRunner{urls: map[string]string{
					dir: tt.url,
				}},
			}

			_, err := r.Resolve(tt.owner, tt.repo)
			matched := err == nil

			if matched != tt.wantOK {
				t.Errorf("originMatches(%q, %q, %q) = %v, want %v (err=%v)",
					tt.url, tt.owner, tt.repo, matched, tt.wantOK, err)
			}
		})
	}
}
