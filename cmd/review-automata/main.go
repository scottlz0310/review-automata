package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
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

	handler := buildHandler(rsv)

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
func buildHandler(rsv *resolver.Resolver) mail.MessageHandler {
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
			return fmt.Errorf("STOP (checkout 失敗): %w", err)
		}

		cleaned := parser.CleanBody(body)
		fmt.Fprintf(os.Stderr, "情報: PR #%d (%s/%s) の処理完了\n", meta.Number, meta.Owner, meta.Repo)
		// TODO: executor に cleaned を渡す（v0.5.0）
		_ = cleaned

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
