package main

import (
	"fmt"
	"io"
)

// Mailer delivers an email. Adapters decide how (stdout, SMTP, ...).
type Mailer interface {
	Send(to, subject, body string) error
}

// StdoutMailer logs emails instead of sending them; used while no SMTP is configured.
type StdoutMailer struct{ Out io.Writer }

func (m StdoutMailer) Send(to, subject, body string) error {
	_, err := fmt.Fprintf(m.Out, "email to=%s subject=%q\n%s\n", to, subject, body)
	return err
}
