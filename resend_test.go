package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestCorrectEmailPersistsAndSendsInviteToNewAddress(t *testing.T) {
	dir := t.TempDir()
	c := newClient(t, dir)
	session, p := openSession(t, c)
	before := len(c.mailer.sent)

	rec := c.do("POST", session+"/email", "de", url.Values{"participant": {"Ben"}, "email": {"ben@new.de"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	sent := c.mailer.sent[before:]
	// The Organizer's language (en, from creation) is used, not the request's.
	if len(sent) != 1 || sent[0].to != "ben@new.de" || !strings.Contains(sent[0].body, "Hello Ben") || !strings.HasSuffix(strings.TrimSpace(sent[0].body), p["Ben"]) {
		t.Fatalf("unexpected mails: %+v", sent)
	}
	page := newClient(t, dir).do("GET", session, "en", nil).Body.String()
	if !strings.Contains(page, "ben@new.de") || strings.Contains(page, "ben@x.de") {
		t.Errorf("email not persisted:\n%s", page)
	}
	for _, n := range []string{"Anna", "Ben", "Cleo"} {
		if !strings.Contains(page, n) {
			t.Errorf("name %s missing", n)
		}
	}
}

func TestInvalidCorrectedEmailIsRejected(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	before := len(c.mailer.sent)
	rec := c.do("POST", session+"/email", "en", url.Values{"participant": {"Ben"}, "email": {"nope"}})
	if rec.Code != http.StatusUnprocessableEntity || len(c.mailer.sent) != before {
		t.Errorf("status %d, mails %d", rec.Code, len(c.mailer.sent)-before)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); !strings.Contains(page, "ben@x.de") {
		t.Errorf("email changed:\n%s", page)
	}
}

func TestResendSendsInviteToCurrentAddress(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	before := len(c.mailer.sent)
	if rec := c.do("POST", session+"/resend", "de", url.Values{"participant": {"Cleo"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "cleo@x.de" || !strings.Contains(sent[0].body, "Hello Cleo") || !strings.HasSuffix(strings.TrimSpace(sent[0].body), p["Cleo"]) {
		t.Fatalf("unexpected mails: %+v", sent)
	}
}

func TestUnknownParticipantIs404(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	for _, action := range []string{"/resend", "/email"} {
		if rec := c.do("POST", session+action, "en", url.Values{"participant": {"Zed"}, "email": {"z@x.de"}}); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", action, rec.Code)
		}
	}
}

func TestSessionPageOffersActionsOnlyBeforeDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	page := c.do("GET", session, "en", nil).Body.String()
	if !strings.Contains(page, session+"/email") || !strings.Contains(page, session+"/resend") {
		t.Errorf("actions missing:\n%s", page)
	}
	submitAll(t, c, p, "en", nil)
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "/email") || strings.Contains(page, "/resend") {
		t.Errorf("actions shown after Draw:\n%s", page)
	}
}

func TestActionsRejectedAfterDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	submitAll(t, c, p, "en", nil)
	before := len(c.mailer.sent)
	for action, f := range map[string]url.Values{
		"/email":  {"participant": {"Ben"}, "email": {"ben@new.de"}},
		"/resend": {"participant": {"Ben"}},
	} {
		if rec := c.do("POST", session+action, "en", f); rec.Code != http.StatusConflict {
			t.Errorf("%s: status %d", action, rec.Code)
		}
	}
	if len(c.mailer.sent) != before {
		t.Errorf("mail sent after Draw")
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "ben@new.de") {
		t.Errorf("email changed after Draw")
	}
}
