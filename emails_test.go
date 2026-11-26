package main

import (
	"strings"
	"testing"
)

func TestEmailRendersEveryLabelledLinkInTextAndHTML(t *testing.T) {
	m := &fakeMailer{}
	a := &app{mailer: m}
	view := mailView{T: catalog["en"], Intro: "Hi", Links: []mailLink{
		{"Session link", "http://x.test/session/s"},
		{"Join link", "http://x.test/join/j"},
	}}
	if err := a.send(Participant{Name: "Anna", Email: "a@x.de"}, "Subj", view); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Session link", "http://x.test/session/s", "Join link", "http://x.test/join/j"} {
		if !strings.Contains(m.sent[0].body, want) {
			t.Errorf("text missing %q:\n%s", want, m.sent[0].body)
		}
		if !strings.Contains(m.sent[0].html, want) {
			t.Errorf("html missing %q:\n%s", want, m.sent[0].html)
		}
	}
}

func TestEmailWithoutLinksHasNoLinkSection(t *testing.T) {
	m := &fakeMailer{}
	a := &app{mailer: m}
	if err := a.send(Participant{Name: "Anna", Email: "a@x.de"}, "Subj", mailView{T: catalog["en"], Intro: "Hi"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.sent[0].html, "<a ") {
		t.Errorf("unexpected link:\n%s", m.sent[0].html)
	}
}
