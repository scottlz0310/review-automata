package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/joho/godotenv"
	"github.com/scottlz0310/review-automata/internal/executor"
	"github.com/scottlz0310/review-automata/internal/git"
	"github.com/scottlz0310/review-automata/internal/mail"
	"github.com/scottlz0310/review-automata/internal/parser"
	"github.com/scottlz0310/review-automata/internal/resolver"
)

func main() {
	// .env ファイルを読み込む（存在しない場合は無視）
	_ = godotenv.Load()

	cfg := mail.Config{
		Addr:     envOrDefault("MAIL_IMAP_ADDR", "imap.gmail.com:993"),
		Username: os.Getenv("MAIL_USERNAME"),
		Password: os.Getenv("MAIL_PASSWORD"),
		Mailbox:  envOrDefault("MAIL_MAILBOX", "INBOX"),
	}

	if cfg.Username == "" || cfg.Password == "" {
		fmt.Fprintln(os.Stderr, "エラー: MAIL_USERNAME と MAIL_PASSWORD 環境変数を設定してください")
		fmt.Fprintln(os.Stderr, "ヒント: sample.env をコピーして .env を作成し、値を設定してください")
		os.Exit(1)
	}

	rsv, err := resolver.New(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: リゾルバー初期化失敗: %v\n", err)
		os.Exit(1)
	}

	exc, err := executor.New(executor.ExecProcessManager{}, executor.ExecCLIRunner{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "エラー: executor の初期化失敗: %v\n", err)
		os.Exit(1)
	}
	handler := buildHandler(rsv, exc)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Fprintf(os.Stderr, "情報: review-automata を起動しました (IMAP: %s, Mailbox: %s)\n", cfg.Addr, cfg.Mailbox)

	if err := mail.New(cfg).Watch(ctx, handler); err != nil {
		if ctx.Err() != nil {
			// シグナルによる正常終了
			fmt.Fprintln(os.Stderr, "情報: シグナルを受信しました。終了します。")
			return
		}
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}
}

// buildHandler は mail.MessageHandler を構築します。
// パース失敗・リポジトリ未解決・checkout 失敗は STOP 条件として error を返しますが、
// ループは継続します（次のメールを処理します）。
func buildHandler(rsv *resolver.Resolver, exc *executor.Executor) mail.MessageHandler {
	return func(subject, body string) error {
		meta, err := parser.ParseSubject(subject)
		if err != nil {
			return fmt.Errorf("STOP (subject パース失敗): %w", err)
		}

		repoPath, err := rsv.Resolve(meta.Owner, meta.Repo)
		if err != nil {
			return fmt.Errorf("STOP (リポジトリ解決失敗): %w", err)
		}

		if err := git.FetchAndCheckout(repoPath, meta.Number, git.ExecCommander{}); err != nil {
			if !errors.Is(err, git.ErrBranchExists) {
				return fmt.Errorf("STOP (checkout 失敗): %w", err)
			}
			// ブランチ既存 → エージェント起動判定（判定不能時も安全側＝確認あり）→ 強制更新
			agentRunning, agentErr := exc.IsAgentRunning()
			if agentErr != nil {
				fmt.Fprintf(os.Stderr, "警告: エージェント起動確認に失敗しました: %v\n", agentErr)
			}
			prompt := fmt.Sprintf("警告: PR #%d のブランチが既に存在します。強制更新して処理しますか? [y/N]: ", meta.Number)
			if agentRunning {
				prompt = fmt.Sprintf("警告: PR #%d のブランチが既に存在し、エージェント CLI が起動中です。強制更新して処理しますか? [y/N]: ", meta.Number)
			}
			fmt.Fprint(os.Stderr, prompt)
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				if scanErr := scanner.Err(); scanErr != nil {
					return fmt.Errorf("STOP (ユーザー入力読み取り失敗): %w", scanErr)
				}
				return fmt.Errorf("STOP (ユーザー入力なし): PR #%d のブランチ強制更新をスキップしました", meta.Number)
			}
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" {
				return fmt.Errorf("STOP (ユーザーキャンセル): PR #%d のブランチ強制更新をスキップしました", meta.Number)
			}
			if agentRunning {
				if killErr := exc.KillAgent(); killErr != nil {
					return fmt.Errorf("STOP (エージェント終了失敗): %w", killErr)
				}
				fmt.Fprintln(os.Stderr, "情報: エージェント CLI の終了を要求しました")
			}
			if err := git.ForceUpdate(repoPath, meta.Number, git.ExecCommander{}); err != nil {
				return fmt.Errorf("STOP (checkout 失敗): %w", err)
			}
		}

		cleaned := parser.CleanBody(body)
		if err := exc.Run(meta.Owner, meta.Repo, meta.Number, cleaned, repoPath); err != nil {
			return fmt.Errorf("STOP (executor 失敗): %w", err)
		}
		fmt.Fprintf(os.Stderr, "情報: PR #%d (%s/%s) の処理完了\n", meta.Number, meta.Owner, meta.Repo)

		return nil
	}
}

// envOrDefault は環境変数 key の値を返します。未設定の場合は fallback を返します。
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
