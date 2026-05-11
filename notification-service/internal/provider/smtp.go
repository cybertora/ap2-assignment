package provider

import (
	"context"
	"fmt"
	"log"
	"net/smtp"

	"notification-service/internal/domain"
)

// SMTPSender — реальная реализация: отправка письма через SMTP.
// Совместим с Mailtrap / Gmail / Mailjet SMTP.
type SMTPSender struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPSender(host, port, user, pass, from string) *SMTPSender {
	return &SMTPSender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *SMTPSender) Name() string { return "SMTP" }

func (s *SMTPSender) Send(ctx context.Context, n domain.Notification) error {
	addr := s.host + ":" + s.port
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)

	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		s.from, n.To, n.Subject, n.Body,
	))

	// net/smtp синхронный — закрываем контекстом через канал.
	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, s.from, []string{n.To}, msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err != nil {
			log.Printf("[PROVIDER SMTP] send failed payment_id=%s err=%v", n.PaymentID, err)
			return err
		}
		log.Printf("[PROVIDER SMTP] ✉️  sent to=%s payment_id=%s", n.To, n.PaymentID)
		return nil
	}
}
