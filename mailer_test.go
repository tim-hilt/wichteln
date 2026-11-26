package main

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
)

// smtpSession is what the fake SMTP server received in one connection.
type smtpSession struct{ auth, from, rcpt, data string }

// fakeSMTPServer accepts one plain-text SMTP connection on localhost, which
// advertises AUTH PLAIN but no STARTTLS, and reports what it received.
func fakeSMTPServer(t *testing.T) (host, port string, got <-chan smtpSession) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	ch := make(chan smtpSession, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		var s smtpSession
		tp.PrintfLine("220 fake ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			verb, _, _ := strings.Cut(line, " ")
			switch strings.ToUpper(verb) {
			case "EHLO":
				tp.PrintfLine("250-fake\r\n250 AUTH PLAIN")
			case "AUTH":
				s.auth = line
				tp.PrintfLine("235 ok")
			case "MAIL":
				s.from = line
				tp.PrintfLine("250 ok")
			case "RCPT":
				s.rcpt = line
				tp.PrintfLine("250 ok")
			case "DATA":
				tp.PrintfLine("354 go ahead")
				data, _ := tp.ReadDotBytes()
				s.data = string(data)
				tp.PrintfLine("250 ok")
			case "QUIT":
				tp.PrintfLine("221 bye")
				ch <- s
				return
			default:
				tp.PrintfLine("502 unknown")
			}
		}
	}()
	host, port, _ = net.SplitHostPort(ln.Addr().String())
	return host, port, ch
}

func TestSMTPMailerDeliversMultipartMail(t *testing.T) {
	host, port, got := fakeSMTPServer(t)
	from, _ := mail.ParseAddress("Wichteln <wichteln@example.test>")
	m := SMTPMailer{Host: host, Port: port, Username: "user", Password: "secret", From: from}
	err := m.Send(Mail{To: "anna@x.de", Subject: "Einladung zum Wichteln ✓", Text: "Hallo Anna,\nschön!", HTML: "<p>Hallo Anna, schön!</p>"})
	if err != nil {
		t.Fatal(err)
	}
	s := <-got

	creds, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(s.auth, "AUTH PLAIN "))
	if string(creds) != "\x00user\x00secret" {
		t.Errorf("auth %q", s.auth)
	}
	if s.from != "MAIL FROM:<wichteln@example.test>" || !strings.HasPrefix(s.rcpt, "RCPT TO:<anna@x.de>") {
		t.Errorf("envelope from=%q rcpt=%q", s.from, s.rcpt)
	}

	msg, err := mail.ReadMessage(strings.NewReader(s.data))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	sender, _ := msg.Header.AddressList("From")
	if subject != "Einladung zum Wichteln ✓" || len(sender) != 1 || *sender[0] != *from || msg.Header.Get("To") != "anna@x.de" {
		t.Errorf("headers %v", msg.Header)
	}
	if msg.Header.Get("Date") == "" || msg.Header.Get("Message-Id") == "" {
		t.Errorf("missing Date or Message-ID: %v", msg.Header)
	}

	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if mediaType != "multipart/alternative" {
		t.Fatalf("content type %q", mediaType)
	}
	parts := map[string]string{}
	r := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(part)
		parts[part.Header.Get("Content-Type")] = string(body)
	}
	if got := strings.ReplaceAll(parts["text/plain; charset=utf-8"], "\r\n", "\n"); got != "Hallo Anna,\nschön!" {
		t.Errorf("text part %q", got)
	}
	if got := parts["text/html; charset=utf-8"]; got != "<p>Hallo Anna, schön!</p>" {
		t.Errorf("html part %q", got)
	}
}

func TestMailerFromEnv(t *testing.T) {
	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}

	if m, err := mailerFromEnv(env(nil)); err != nil {
		t.Errorf("no SMTP_HOST: err %v", err)
	} else if _, ok := m.(StdoutMailer); !ok {
		t.Errorf("no SMTP_HOST: want StdoutMailer, got %T", m)
	}

	for _, from := range []string{"", "not an address"} {
		if _, err := mailerFromEnv(env(map[string]string{"SMTP_HOST": "smtp.example.test", "MAIL_FROM": from})); err == nil {
			t.Errorf("MAIL_FROM %q: want error", from)
		}
	}

	m, err := mailerFromEnv(env(map[string]string{
		"SMTP_HOST": "smtp.example.test", "SMTP_USER": "u", "SMTP_PASSWORD": "p",
		"MAIL_FROM": "Wichteln <wichteln@example.test>",
	}))
	s, ok := m.(SMTPMailer)
	if err != nil || !ok || s.Host != "smtp.example.test" || s.Port != "465" || s.Username != "u" || s.Password != "p" || s.From.Address != "wichteln@example.test" {
		t.Errorf("got %+v, %v", m, err)
	}
}
