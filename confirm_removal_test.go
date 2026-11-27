package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// breakingSession is a Session of Anna, Ben, Cleo (all joined) and Dora (not
// joined). Anna and Ben exclude each other, Cleo submitted too. Removing Dora
// leaves no valid Draw; only Ben's Exclusion has to go.
func breakingSession(t *testing.T) (c *client, session string, p map[string]string) {
	t.Helper()
	c = newClient(t, t.TempDir())
	session, p = openSession(t, c)
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a", "Ben"))
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "b", "Anna"))
	return c, session, p
}

func confirmNames(c *client, session, lang string, affected []string, names ...string) *httptest.ResponseRecorder {
	v := url.Values{"name": names, "affected": affected}
	return c.do("POST", session+"/names", lang, v)
}

func TestRemovalBreakingTheDrawAsksForConfirmationAndSavesNothing(t *testing.T) {
	c, session, p := breakingSession(t)
	before := len(c.mailer.sent)
	rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo")
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `name="affected" value="Ben"`) {
		t.Fatalf("status %d, no confirmation:\n%s", rec.Code, body)
	}
	if strings.Contains(body, `name="affected" value="Anna"`) || strings.Contains(body, "excludes") {
		t.Errorf("confirmation leaks Exclusions:\n%s", body)
	}
	if !strings.Contains(c.do("GET", session, "en", nil).Body.String(), "Dora") {
		t.Error("Dora removed without confirmation")
	}
	if len(c.mailer.sent) != before {
		t.Error("emails sent without confirmation")
	}
	if !strings.Contains(c.do("GET", p["Ben"], "en", nil).Body.String(), `value="Anna" checked`) {
		t.Error("Ben's Exclusion dropped without confirmation")
	}
}

func TestConfirmedRemovalDropsExclusionsUnsubmitsAndEmailsAffected(t *testing.T) {
	c, session, p := breakingSession(t)
	before := len(c.mailer.sent)
	if rec := confirmNames(c, session, "en", []string{"Ben"}, "Anna", "Ben", "Cleo"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d:\n%s", rec.Code, rec.Body.String())
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "Dora") || strings.Contains(page, "have been drawn") {
		t.Errorf("Dora still there, or drawn too early:\n%s", page)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "ben@x.de" || !strings.Contains(sent[0].body, "Dora") ||
		!strings.Contains(sent[0].body, "http://example.test"+p["Ben"]) {
		t.Fatalf("want one email to Ben naming Dora with his link: %+v", sent)
	}
	if page := c.do("GET", p["Ben"], "en", nil).Body.String(); strings.Contains(page, `value="Anna" checked`) {
		t.Errorf("Ben's Exclusion not dropped:\n%s", page)
	}
	// Anna kept her Exclusion; once Ben confirms again the Draw runs.
	c.do("GET", p["Anna"], "en", nil) // dismiss the Join confirmation
	if !strings.Contains(c.do("GET", p["Anna"], "en", nil).Body.String(), `value="Ben" checked`) {
		t.Error("Anna's Exclusion dropped")
	}
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "b"))
	if page := c.do("GET", session, "en", nil).Body.String(); !strings.Contains(page, "The names have been drawn.") {
		t.Errorf("not drawn after Ben confirmed:\n%s", page)
	}
}

func TestConfirmWithChangedAffectedSetShowsConfirmationAgain(t *testing.T) {
	c, session, _ := breakingSession(t)
	rec := confirmNames(c, session, "en", []string{"Cleo"}, "Anna", "Ben", "Cleo")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `name="affected" value="Ben"`) {
		t.Errorf("status %d, no new confirmation:\n%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(c.do("GET", session, "en", nil).Body.String(), "Dora") {
		t.Error("Dora removed despite mismatch")
	}
}

func TestRemovalConfirmationAndEmailInGerman(t *testing.T) {
	c, session, _ := breakingSession(t)
	if body := saveNames(c, session, "de", "Anna", "Ben", "Cleo").Body.String(); !strings.Contains(body, "Bestätigen") {
		t.Errorf("confirmation not German:\n%s", body)
	}
	before := len(c.mailer.sent)
	confirmNames(c, session, "de", []string{"Ben"}, "Anna", "Ben", "Cleo")
	// Ben's last Submission was in English.
	if m := c.mailer.sent[before]; !strings.Contains(m.subject, "Secret Santa") {
		t.Errorf("email not in Ben's language: %+v", m)
	}
}
