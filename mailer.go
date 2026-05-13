package fusionsolar

import (
	"io"
	"log/slog"
	"net"
	"strconv"

	"gopkg.in/gomail.v2"
)

type Mailer struct {
	SmtpUser         string
	SmtpFrom         string
	SmtpPasswd       string
	SmtpServer       string
	SmtpDestinations []string
}

type Attachment struct {
	Name    string
	Content []byte
}

func (m *Mailer) SendEmail(subject, body string, attachments []Attachment) {
	slog.Info("[*] Enviando emails")

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.SmtpFrom)
	msg.SetHeader("To", m.SmtpDestinations...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", body)

	for _, att := range attachments {
		msg.Attach(att.Name, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(att.Content)
			return err
		}))
	}

	host, portStr, err := net.SplitHostPort(m.SmtpServer)
	if err != nil {
		// m.SmtpServer might not have a port, try to append default
		host = m.SmtpServer
		portStr = "587"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		ReportError("invalid smtp port", err)
		return
	}

	d := gomail.NewDialer(host, port, m.SmtpUser, m.SmtpPasswd)

	if err := d.DialAndSend(msg); err != nil {
		ReportError("Error sending email", err)
	}
}
