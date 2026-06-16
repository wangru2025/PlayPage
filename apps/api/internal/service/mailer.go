package service

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Mailer struct {
	host      string
	port      string
	username  string
	password  string
	fromEmail string
}

func NewMailer(host, port, username, password, fromEmail string) *Mailer {
	return &Mailer{
		host:      strings.TrimSpace(host),
		port:      strings.TrimSpace(port),
		username:  strings.TrimSpace(username),
		password:  password,
		fromEmail: strings.TrimSpace(fromEmail),
	}
}

func (m *Mailer) IsNoop() bool {
	return m.host == "" || m.fromEmail == ""
}

func (m *Mailer) SendCode(to, code string) error {
	return m.SendText(to, "PlayPage 登录验证码", fmt.Sprintf("你的 PlayPage 登录验证码是 %s，10 分钟内有效。", code))
}

func (m *Mailer) SendText(to, subject, body string) error {
	to = strings.TrimSpace(to)
	start := time.Now()
	maskedTo := maskMailAddress(to)
	if m.IsNoop() {
		log.Printf("mail_send_disabled to=%s subject=%q", maskedTo, subject)
		return nil
	}
	if to == "" {
		return fmt.Errorf("mail recipient is empty")
	}
	log.Printf("mail_send_start to=%s subject=%q smtp_host=%s", maskedTo, subject, m.host)

	conn, err := net.Dial("tcp", net.JoinHostPort(m.host, m.port))
	if err != nil {
		return m.mailError("dial_smtp", maskedTo, subject, start, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return m.mailError("create_smtp_client", maskedTo, subject, start, err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return m.mailError("starttls", maskedTo, subject, start, err)
		}
	}

	if m.username != "" && m.password != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
				return m.mailError("smtp_auth", maskedTo, subject, start, err)
			}
		}
	}

	if err := client.Mail(m.fromEmail); err != nil {
		return m.mailError("mail_from", maskedTo, subject, start, err)
	}
	if err := client.Rcpt(to); err != nil {
		return m.mailError("mail_rcpt", maskedTo, subject, start, err)
	}

	writer, err := client.Data()
	if err != nil {
		return m.mailError("mail_data", maskedTo, subject, start, err)
	}
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nDate: %s\r\nMessage-ID: <%s>\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\nX-Mailer: PlayPage Mailer\r\n\r\n%s\r\n",
		m.fromEmail,
		to,
		time.Now().Format(time.RFC1123Z),
		m.messageID(),
		mime.QEncoding.Encode("utf-8", subject),
		body,
	)
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return m.mailError("write_mail_body", maskedTo, subject, start, err)
	}
	if err := writer.Close(); err != nil {
		return m.mailError("close_mail_body", maskedTo, subject, start, err)
	}
	if err := client.Quit(); err != nil {
		return m.mailError("quit_smtp_client", maskedTo, subject, start, err)
	}
	log.Printf("mail_send_ok to=%s subject=%q duration=%s", maskedTo, subject, time.Since(start))
	return nil
}

func (m *Mailer) mailError(stage, maskedTo, subject string, start time.Time, err error) error {
	wrapped := fmt.Errorf("%s: %w", strings.ReplaceAll(stage, "_", " "), err)
	log.Printf("mail_send_failed stage=%s to=%s subject=%q duration=%s err=%v", stage, maskedTo, subject, time.Since(start), err)
	return wrapped
}

func (m *Mailer) messageID() string {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return fmt.Sprintf("%d@%s", time.Now().UnixNano(), m.host)
	}
	host := m.host
	if host == "" {
		host = "playpage.local"
	}
	return fmt.Sprintf("%d.%s@%s", time.Now().UnixNano(), hex.EncodeToString(random[:]), host)
}

func maskMailAddress(address string) string {
	address = strings.TrimSpace(address)
	at := strings.LastIndex(address, "@")
	if at <= 0 {
		return "***"
	}
	name := address[:at]
	domain := address[at+1:]
	if len(name) <= 2 {
		return name[:1] + "***@" + domain
	}
	return name[:2] + "***@" + domain
}
