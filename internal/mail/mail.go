// Package mail は IMAP IDLE ベースのメール監視機能（PoC）を提供します。
// 本番実装では Gmail API + Cloud Pub/Sub を使用する予定です。
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	gomail "github.com/emersion/go-message/mail"
)

// Config は IMAP 接続設定を保持します。
type Config struct {
	Addr     string // ホスト:ポート (例: "imap.gmail.com:993")
	Username string // メールアドレス (例: "user@gmail.com")
	Password string // App Password（コードへのハードコード禁止・環境変数で渡すこと）
	Mailbox  string // 監視するメールボックス (デフォルト: "INBOX")
}

// validate は設定値を検証します。Mailbox が空の場合は "INBOX" を補完します。
func (c *Config) validate() error {
	if c.Addr == "" {
		return fmt.Errorf("IMAP アドレスが未設定です")
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return fmt.Errorf("IMAP アドレスの形式が不正です（\"ホスト:ポート\" 形式で指定してください、例: \"imap.gmail.com:993\"）: %w", err)
	}
	if c.Username == "" {
		return fmt.Errorf("ユーザー名が未設定です")
	}
	if c.Password == "" {
		return fmt.Errorf("パスワードが未設定です")
	}
	if c.Mailbox == "" {
		c.Mailbox = "INBOX"
	}
	return nil
}

// MessageHandler は新着メールの subject と body を受け取るコールバックです。
// エラーを返した場合はログに記録しますが、監視ループは継続します（STOP 条件でも同様）。
type MessageHandler func(subject, body string) error

// Watcher は IMAP IDLE でメールボックスを監視します。
type Watcher struct {
	cfg Config
}

// New は Watcher を生成します。
func New(cfg Config) *Watcher {
	return &Watcher{cfg: cfg}
}

// Watch は IDLE でメールボックスを監視し、新着メールを handler に渡します。
// ctx がキャンセルされると終了します。
// 設定エラー・handler 未設定・接続失敗は即座に error を返します。
func (w *Watcher) Watch(ctx context.Context, handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("ハンドラーが未設定です")
	}
	if err := w.cfg.validate(); err != nil {
		return fmt.Errorf("IMAP 設定エラー: %w", err)
	}

	c, err := w.connect()
	if err != nil {
		return err
	}
	defer func() { _ = c.Logout() }()

	// 起動時の未読メッセージを処理
	if err := w.fetchAndProcess(c, handler); err != nil {
		fmt.Fprintf(os.Stderr, "初期メッセージ処理エラー: %v\n", err)
	}

	return w.idleLoop(ctx, c, handler)
}

// connect は TLS で IMAP サーバーに接続し、ログイン・メールボックス選択を行います。
func (w *Watcher) connect() (*imapClient.Client, error) {
	host := w.cfg.Addr
	if h, _, err := net.SplitHostPort(w.cfg.Addr); err == nil {
		host = h
	}

	c, err := imapClient.DialTLS(w.cfg.Addr, &tls.Config{ServerName: host})
	if err != nil {
		return nil, fmt.Errorf("IMAP TLS 接続失敗 (%s): %w", w.cfg.Addr, err)
	}

	if err := c.Login(w.cfg.Username, w.cfg.Password); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("IMAP ログイン失敗: %w", err)
	}

	if _, err := c.Select(w.cfg.Mailbox, false); err != nil {
		_ = c.Logout()
		return nil, fmt.Errorf("メールボックス選択失敗 (%s): %w", w.cfg.Mailbox, err)
	}

	return c, nil
}

// idleLoop は IDLE でメールボックスを監視します。
// Gmail の IDLE タイムアウト（29 分）を考慮し、15 分ごとにリフレッシュします。
// MailboxUpdate を受信した場合のみ IDLE を停止してフェッチします。
// MailboxUpdate 以外の更新（Expunge 等）は IDLE を継続したまま待機し続けます。
func (w *Watcher) idleLoop(ctx context.Context, c *imapClient.Client, handler MessageHandler) error {
	updates := make(chan imapClient.Update, 10)
	c.Updates = updates

	const refreshInterval = 15 * time.Minute

outerLoop:
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		stop := make(chan struct{})
		idleDone := make(chan error, 1)
		go func() {
			idleDone <- c.Idle(stop, nil)
		}()

		timer := time.NewTimer(refreshInterval)
		shouldFetch := false

	waitLoop:
		for {
			select {
			case update, ok := <-updates:
				if !ok {
					// updates チャネルクローズ → 接続切断
					if !timer.Stop() {
						<-timer.C
					}
					close(stop)
					<-idleDone
					return fmt.Errorf("IMAP 接続が切断されました（更新チャネルクローズ）")
				}
				if _, ok2 := update.(*imapClient.MailboxUpdate); ok2 {
					shouldFetch = true
				}
				// 残りの pending updates をドレインし、まとめて判定する
			drainLoop:
				for {
					select {
					case u, ok := <-updates:
						if !ok {
							if !timer.Stop() {
								<-timer.C
							}
							close(stop)
							<-idleDone
							return fmt.Errorf("IMAP 接続が切断されました（更新チャネルクローズ）")
						}
						if _, ok2 := u.(*imapClient.MailboxUpdate); ok2 {
							shouldFetch = true
						}
					default:
						break drainLoop
					}
				}
				if shouldFetch {
					// MailboxUpdate あり → IDLE を停止してフェッチへ
					if !timer.Stop() {
						<-timer.C
					}
					break waitLoop
				}
				// MailboxUpdate なし → IDLE を継続して次の更新を待機
			case <-timer.C:
				// リフレッシュタイムアウト → IDLE を停止
				break waitLoop
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				close(stop)
				<-idleDone
				return ctx.Err()
			case err := <-idleDone:
				if !timer.Stop() {
					<-timer.C
				}
				if err != nil {
					return fmt.Errorf("IDLE エラー: %w", err)
				}
				continue outerLoop
			}
		}

		close(stop)
		if err := <-idleDone; err != nil {
			return fmt.Errorf("IDLE 停止エラー: %w", err)
		}

		if shouldFetch {
			if err := w.fetchAndProcess(c, handler); err != nil {
				fmt.Fprintf(os.Stderr, "メッセージ処理エラー: %v\n", err)
			}
		}
	}
}

// fetchAndProcess は UNSEEN メッセージを取得し handler に渡します。
// 本日（起動日）に受信したメッセージのみを対象とし、過去の未読メールは処理しません。
// handler が成功したメッセージのみ SEEN フラグを設定します（失敗時は未読のまま残り再処理可能）。
func (w *Watcher) fetchAndProcess(c *imapClient.Client, handler MessageHandler) error {
	// 本日0時（ローカル時刻）以降のメールのみを対象にする
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	criteria.Since = today

	nums, err := c.Search(criteria)
	if err != nil {
		return fmt.Errorf("メッセージ検索失敗: %w", err)
	}
	if len(nums) == 0 {
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(nums...)

	section := &imap.BodySectionName{}
	messages := make(chan *imap.Message, 10) // 固定バッファでメモリ使用量を抑制
	fetchDone := make(chan error, 1)
	go func() {
		fetchDone <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}, messages)
	}()

	var toMark []uint32
	for msg := range messages {
		if msg == nil {
			continue
		}
		subject := ""
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
		}
		body := extractTextBody(msg, section)

		if handlerErr := handler(subject, body); handlerErr != nil {
			fmt.Fprintf(os.Stderr, "ハンドラーエラー (subject: %q): %v\n", subject, handlerErr)
		} else {
			// handler が成功した場合のみ SEEN マーク対象に追加する
			toMark = append(toMark, msg.SeqNum)
		}
	}

	if err := <-fetchDone; err != nil {
		return fmt.Errorf("メッセージ取得失敗: %w", err)
	}

	// 処理済みメッセージに SEEN フラグを設定
	if len(toMark) > 0 {
		markSet := new(imap.SeqSet)
		markSet.AddNum(toMark...)
		item := imap.FormatFlagsOp(imap.AddFlags, true)
		if err := c.Store(markSet, item, []interface{}{imap.SeenFlag}, nil); err != nil {
			return fmt.Errorf("SEEN フラグ設定失敗: %w", err)
		}
	}

	return nil
}

// extractTextBody は IMAP メッセージから text/plain 部分を抽出します。
// MIME リーダーが完全に生成できなかった場合は raw ボディを返します。
// text/plain パートが見つからない場合は空文字を返します。
func extractTextBody(msg *imap.Message, section *imap.BodySectionName) string {
	r := msg.GetBody(section)
	if r == nil {
		return ""
	}

	mr, err := gomail.CreateReader(r)
	if err != nil {
		// 不明な charset でも mr は使用可能。パース不能な場合はフォールバック。
		if mr == nil {
			b, _ := io.ReadAll(r)
			return string(b)
		}
	}

	for {
		p, err := mr.NextPart()
		if err != nil {
			break
		}
		if h, ok := p.Header.(*gomail.InlineHeader); ok {
			ct, _, _ := h.ContentType()
			if ct == "text/plain" {
				b, _ := io.ReadAll(p.Body)
				return string(b)
			}
		}
	}

	return ""
}
