package main

import (
	"bytes"
	"fmt"
	"strings"
)

// mailLink is a labelled URL in an email, e.g. "Join link".
type mailLink struct{ Label, URL string }

// mailView is the content of an email to one Participant. The plain-text and
// the HTML version (web/templates/email.html) are rendered from it.
type mailView struct {
	T        Messages
	Greeting string
	Intro    string     // paragraphs separated by a blank line
	Links    []mailLink // zero or more, each shown as a button
	Wish     string     // optional, the recipient's Wish
}

// Paras returns Intro split into paragraphs.
func (v mailView) Paras() []string { return strings.Split(v.Intro, "\n\n") }

// send emails the view to the Participant, greeting them by name.
func (a *app) send(p Participant, subject string, v mailView) error {
	v.Greeting = fmt.Sprintf(v.T["mail_greeting"], p.Name)
	var text strings.Builder
	fmt.Fprintf(&text, "%s\n\n%s\n", v.Greeting, v.Intro)
	for _, l := range v.Links {
		fmt.Fprintf(&text, "\n%s:\n%s\n", l.Label, l.URL)
	}
	if v.Wish != "" {
		fmt.Fprintf(&text, "\n%s:\n%s\n", v.T["draw_wish"], v.Wish)
	}
	var html bytes.Buffer
	if err := templates.ExecuteTemplate(&html, "email.html", v); err != nil {
		return err
	}
	return a.mailer.Send(Mail{To: p.Email, Subject: subject, Text: text.String(), HTML: html.String()})
}
