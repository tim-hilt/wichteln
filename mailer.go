package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

// Mail is one email with a plain-text and an HTML version of the same content.
type Mail struct{ To, Subject, Text, HTML string }

// Mailer delivers an email. Adapters decide how (stdout, SMTP, ...).
type Mailer interface {
	Send(Mail) error
}

// mailerFromEnv returns an SMTPMailer if SMTP_HOST is set, otherwise a
// StdoutMailer, so the app runs without a mail provider.
func mailerFromEnv(getenv func(string) string) (Mailer, error) {
	host := getenv("SMTP_HOST")
	if host == "" {
		return StdoutMailer{Out: os.Stdout}, nil
	}
	from, err := mail.ParseAddress(getenv("MAIL_FROM"))
	if err != nil {
		return nil, fmt.Errorf("MAIL_FROM: %w", err)
	}
	port := getenv("SMTP_PORT")
	if port == "" {
		port = "465"
	}
	return SMTPMailer{host, port, getenv("SMTP_USER"), getenv("SMTP_PASSWORD"), from}, nil
}

// StdoutMailer logs emails instead of sending them; used while no SMTP is configured.
type StdoutMailer struct{ Out io.Writer }

func (m StdoutMailer) Send(mail Mail) error {
	_, err := fmt.Fprintf(m.Out, "email to=%s subject=%q\n%s\n", mail.To, mail.Subject, mail.Text)
	return err
}

// SMTPMailer submits emails to an SMTP server. Port 465 uses implicit TLS;
// any other port upgrades with STARTTLS when the server offers it. Credentials
// are only ever sent encrypted (or to localhost).
type SMTPMailer struct {
	Host, Port         string
	Username, Password string
	From               *mail.Address
}

// smtpTimeout bounds a whole delivery, so a hanging server can't block a request.
const smtpTimeout = 30 * time.Second

func (m SMTPMailer) Send(mail Mail) error {
	msg, err := m.message(mail)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(m.Host, m.Port)
	dialer := &net.Dialer{Timeout: smtpTimeout}
	var conn net.Conn
	if m.Port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: m.Host})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(smtpTimeout))

	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.Host}); err != nil {
			return err
		}
	}
	if m.Username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
			return err
		}
	}
	if err := c.Mail(m.From.Address); err != nil {
		return err
	}
	if err := c.Rcpt(mail.To); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// message renders mail as a multipart/alternative MIME message.
func (m SMTPMailer) message(mail Mail) ([]byte, error) {
	var buf bytes.Buffer
	body := multipart.NewWriter(&buf)
	fmt.Fprintf(&buf, "From: %s\r\n", m.From)
	fmt.Fprintf(&buf, "To: %s\r\n", mail.To)
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", mail.Subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: <%s@%s>\r\n", rand.Text(), m.From.Address[strings.LastIndex(m.From.Address, "@")+1:])
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", body.Boundary())
	for _, p := range []struct{ contentType, content string }{
		{"text/plain; charset=utf-8", mail.Text},
		{"text/html; charset=utf-8", mail.HTML},
	} {
		part, err := body.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {p.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		qp := quotedprintable.NewWriter(part)
		if _, err := io.WriteString(qp, p.content); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
	}
	if err := body.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
