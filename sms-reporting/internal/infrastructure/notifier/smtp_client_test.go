package notifier

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runDummySMTPServer(t *testing.T, listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Initial greeting
	_, _ = writer.WriteString("220 mock.smtp.server ESMTP\r\n")
	writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
			_, _ = writer.WriteString("250-mock.smtp.server\r\n250 AUTH PLAIN\r\n")
			writer.Flush()
		} else if strings.HasPrefix(line, "AUTH PLAIN") {
			_, _ = writer.WriteString("235 2.7.0 Authentication successful\r\n")
			writer.Flush()
		} else if strings.HasPrefix(line, "MAIL FROM") {
			_, _ = writer.WriteString("250 2.1.0 Ok\r\n")
			writer.Flush()
		} else if strings.HasPrefix(line, "RCPT TO") {
			_, _ = writer.WriteString("250 2.1.5 Ok\r\n")
			writer.Flush()
		} else if strings.HasPrefix(line, "DATA") {
			_, _ = writer.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
			writer.Flush()
			
			// Read until \r\n.\r\n
			for {
				dataLine, _ := reader.ReadString('\n')
				if strings.TrimSpace(dataLine) == "." {
					_, _ = writer.WriteString("250 2.0.0 Ok: queued\r\n")
					writer.Flush()
					break
				}
			}
		} else if strings.HasPrefix(line, "QUIT") {
			_, _ = writer.WriteString("221 2.0.0 Bye\r\n")
			writer.Flush()
			return
		}
	}
}

func TestSMTPMailer_SendReportEmail(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	go runDummySMTPServer(t, listener)

	host, port, _ := net.SplitHostPort(listener.Addr().String())

	cfg := Config{
		Host:     host,
		Port:     port,
		UseAuth:  true,
		UseTLS:   false,
		Username: "user",
		Password: "password",
		From:     "sender@example.com",
		FromName: "Sender Name",
	}

	mailer := NewMailer(cfg)

	err = mailer.SendReportEmail(context.Background(), "recipient@example.com", "Test Subject", "<h1>Test</h1>")
	assert.NoError(t, err)
}

func TestSMTPMailer_Ping_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	go runDummySMTPServer(t, listener)

	host, port, _ := net.SplitHostPort(listener.Addr().String())

	err = Ping(context.Background(), host, port)
	assert.NoError(t, err)
}

func TestSMTPMailer_Ping_Failure(t *testing.T) {
	err := Ping(context.Background(), "127.0.0.1", "1") // Assuming port 1 is closed
	assert.Error(t, err)
}

func TestSMTPMailer_SendReportEmail_DialError(t *testing.T) {
	cfg := Config{
		Host: "127.0.0.1",
		Port: "1", // invalid port
	}

	mailer := NewMailer(cfg)

	err := mailer.SendReportEmail(context.Background(), "recipient@example.com", "Test Subject", "<h1>Test</h1>")
	assert.Error(t, err)
}
