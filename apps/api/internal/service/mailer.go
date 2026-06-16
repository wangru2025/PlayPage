package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net"
	"net/smtp"
	"strings"
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
	if m.IsNoop() {
		log.Printf("mail disabled, to=%s subject=%q", to, subject)
		return nil
	}
	if to == "" {
		return fmt.Errorf("mail recipient is empty")
	}

	conn, err := net.Dial("tcp", net.JoinHostPort(m.host, m.port))
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if m.username != "" && m.password != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(m.fromEmail); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mail rcpt: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail data: %w", err)
	}
	message := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		m.fromEmail,
		to,
		mime.QEncoding.Encode("utf-8", subject),
		body,
	)
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write mail body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close mail body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit smtp client: %w", err)
	}
	return nil
}
