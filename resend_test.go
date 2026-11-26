package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestResendToJoinedParticipantSendsConfirmationInTheirLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	loc := c.do("POST", join, "de", joinForm("Cleo", "cleo@x.de", "w")).Header().Get("Location")
	before := len(c.mailer.sent)
	// The request's language (en) is not used, Cleo's (de) is.
	if rec := c.do("POST", session+"/resend", "en", url.Values{"participant": {"Cleo"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "cleo@x.de" || !strings.Contains(sent[0].body, "Hallo Cleo") || !strings.Contains(sent[0].body, "http://example.test"+loc) {
		t.Fatalf("unexpected mails: %+v", sent)
	}
}

func TestResendToOrganizerSendsCompleteEmailInOrganizerLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "de")
	before := len(c.mailer.sent)
	if rec := c.do("POST", session+"/resend", "en", url.Values{"participant": {"Anna"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "anna@x.de" || !strings.Contains(sent[0].body, "Hallo Anna") {
		t.Fatalf("unexpected mails: %+v", sent)
	}
	for _, want := range []string{session, join, anna} {
		if !strings.Contains(sent[0].body, want) {
			t.Errorf("email missing %q:\n%s", want, sent[0].body)
		}
	}
}

func TestResendIsNotAvailableForUnknownOrNotJoinedNames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	before := len(c.mailer.sent)
	for _, name := range []string{"Zed", "Ben"} {
		if rec := c.do("POST", session+"/resend", "en", url.Values{"participant": {name}}); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", name, rec.Code)
		}
	}
	if len(c.mailer.sent) != before {
		t.Error("mail sent")
	}
	page := c.do("GET", session, "en", nil).Body.String()
	if n := strings.Count(page, "Resend link"); n != 1 {
		t.Errorf("want Resend only for Anna, got %d:\n%s", n, page)
	}
}

func TestOrganizerCorrectEmailRouteIsRemoved(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	rec := c.do("POST", session+"/email", "en", url.Values{"participant": {"Anna"}, "email": {"new@x.de"}})
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status %d", rec.Code)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, session+"/email") {
		t.Error("session page still offers email correction")
	}
}

func TestResendOnlyBeforeDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	if page := c.do("GET", session, "en", nil).Body.String(); !strings.Contains(page, session+"/resend") {
		t.Errorf("resend missing:\n%s", page)
	}
	submitAll(t, c, p, "en", nil)
	before := len(c.mailer.sent)
	if rec := c.do("POST", session+"/resend", "en", url.Values{"participant": {"Ben"}}); rec.Code != http.StatusConflict {
		t.Errorf("status %d", rec.Code)
	}
	if len(c.mailer.sent) != before {
		t.Error("mail sent after Draw")
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "/resend") {
		t.Errorf("resend shown after Draw:\n%s", page)
	}
}
