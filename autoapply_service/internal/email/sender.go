package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
)

// SMTPConfig holds SMTP configuration.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Sender handles sending emails.
type Sender struct {
	smtp *SMTPConfig
}

// NewSender creates a new email sender.
func NewSender(smtpConfig *SMTPConfig) *Sender {
	return &Sender{smtp: smtpConfig}
}

// EmailMessage represents an email to send.
type EmailMessage struct {
	To          string
	Subject     string
	Body        string
	Attachments []Attachment
	FromName    string
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename string
	Content  []byte
	MimeType string
}

// Send sends an email using SMTP.
func (s *Sender) Send(ctx context.Context, msg *EmailMessage) error {
	if s.smtp == nil || s.smtp.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	slog.Info("Sending email via SMTP",
		"to", msg.To,
		"subject", msg.Subject,
	)

	// Build email
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Headers
	fromHeader := s.smtp.From
	if msg.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", msg.FromName, s.smtp.From)
	}

	headers := make(textproto.MIMEHeader)
	headers.Set("From", fromHeader)
	headers.Set("To", msg.To)
	headers.Set("Subject", msg.Subject)
	headers.Set("MIME-Version", "1.0")
	headers.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary()))

	var headerBuf bytes.Buffer
	for k, v := range headers {
		headerBuf.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}
	headerBuf.WriteString("\r\n")

	// Body part
	bodyPart, _ := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	bodyPart.Write([]byte(msg.Body))

	// Attachments
	for _, att := range msg.Attachments {
		attPart, _ := writer.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {att.MimeType},
			"Content-Disposition":       {fmt.Sprintf(`attachment; filename="%s"`, att.Filename)},
			"Content-Transfer-Encoding": {"base64"},
		})
		encoded := base64.StdEncoding.EncodeToString(att.Content)
		// Wrap at 76 characters
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attPart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()

	// Combine headers and body
	fullMessage := append(headerBuf.Bytes(), buf.Bytes()...)

	// Send
	auth := smtp.PlainAuth("", s.smtp.Username, s.smtp.Password, s.smtp.Host)
	addr := fmt.Sprintf("%s:%d", s.smtp.Host, s.smtp.Port)

	err := smtp.SendMail(addr, auth, s.smtp.From, []string{msg.To}, fullMessage)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Email sent successfully", "to", msg.To)
	return nil
}

// IsSMTPConfigured checks if SMTP is configured.
func (s *Sender) IsSMTPConfigured() bool {
	return s.smtp != nil && s.smtp.Host != "" && s.smtp.Username != ""
}
