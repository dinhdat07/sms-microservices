package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	appLogger "log"
	"net"
	stdsmtp "net/smtp"
	"strings"
)

// Message represents an email message to be sent.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

type Config struct {
	Host     string
	Port     string
	UseAuth  bool
	UseTLS   bool
	Username string
	Password string

	From     string
	FromName string
}

// SMTPMailer implements domain.ReportNotifier via SMTP.
type SMTPMailer struct {
	cfg Config
}

func NewMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func Ping(ctx context.Context, host string, port string) error {
	addr := net.JoinHostPort(host, port)

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp server %s: %w", addr, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			appLogger.Println("failed to close smtp ping connection", "error", err)
		}
	}()

	appLogger.Println("smtp server connection verified", "address", addr)
	return nil
}

// SendReportEmail implements domain.ReportNotifier interface.
func (m *SMTPMailer) SendReportEmail(ctx context.Context, toEmail string, subject string, htmlBody string) error {
	msg := Message{
		To:       toEmail,
		Subject:  subject,
		HTMLBody: htmlBody,
	}
	return m.send(ctx, msg)
}

func (m *SMTPMailer) send(ctx context.Context, msg Message) error {
	fromHeader := m.cfg.From
	if strings.TrimSpace(m.cfg.FromName) != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mimeHeaderEncode(m.cfg.FromName), m.cfg.From)
	}

	raw := buildMultipartMessage(
		fromHeader,
		msg.To,
		msg.Subject,
		msg.TextBody,
		msg.HTMLBody,
	)

	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)

	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}

	client, err := stdsmtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("create smtp client: %w", err)
	}
	closed := false
	defer func() {
		if closed {
			return
		}
		if err := client.Close(); err != nil {
			appLogger.Println("failed to close smtp client connection", "error", err)
		}
	}()

	if m.cfg.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: m.cfg.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start tls: %w", err)
		}
	}

	if m.cfg.UseAuth {
		auth := stdsmtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write([]byte(raw)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("close message writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	closed = true

	return nil
}
