package notifier

import (
	"bytes"
	"fmt"
	"time"
)

func buildMultipartMessage(from, to, subject, textBody, htmlBody string) string {
	boundary := fmt.Sprintf("portal-boundary-%d", time.Now().UnixNano())

	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", mimeHeaderEncode(subject)))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf(`Content-Type: multipart/alternative; boundary="%s"`+"\r\n", boundary))
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString(`Content-Type: text/plain; charset="UTF-8"` + "\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(textBody)
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString(`Content-Type: text/html; charset="UTF-8"` + "\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buf.String()
}

// simple for local/MailHog
func mimeHeaderEncode(s string) string {
	return s
}
