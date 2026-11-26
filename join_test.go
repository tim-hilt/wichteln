package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func joinForm(name, email, wish string, excluded ...string) url.Values {
	v := wishForm(wish, excluded...)
	v.Set("name", name)
	v.Set("email", email)
	return v
}

func TestSessionPageShowsJoinLinkAndStatusesOnly(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "secret-wish", "Cleo"))
	page := c.do("GET", session, "en", nil).Body.String()
	for _, want := range []string{"http://example.test" + join, "not joined", "joined"} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "secret-wish") {
		t.Error("Wish shown on Session page")
	}
}

func TestJoinPageListsOnlyUnjoinedNames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w"))
	body := c.do("GET", join, "en", nil).Body.String()
	if !strings.Contains(body, "Cleo") || strings.Contains(body, "Ben") || strings.Contains(body, "Anna") {
		t.Errorf("want only Cleo:\n%s", body)
	}
	if strings.Contains(body, session) || strings.Contains(body, "@") {
		t.Errorf("Join page leaks Session details:\n%s", body)
	}
	if !strings.Contains(c.do("GET", join, "de", nil).Body.String(), "Wähle deinen Namen") {
		t.Error("not localized")
	}
}

func TestUnknownJoinLinkIsNotFound(t *testing.T) {
	c := newClient(t, t.TempDir())
	for _, m := range []string{"GET", "POST"} {
		if rec := c.do(m, "/join/"+strings.Repeat("a", 26), "en", url.Values{}); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", m, rec.Code)
		}
	}
}

func TestSelectingNameShowsGreetingFormOrSelectionAgain(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join := openJoinSession(t, c)
	body := c.do("GET", join+"?name=Ben", "en", nil).Body.String()
	for _, want := range []string{"Hello Ben!", `name="email"`, `name="wish"`, `value="Cleo"`, `value="Ben"`} {
		if !strings.Contains(body, want) {
			t.Errorf("form missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, `value="Anna"`) && strings.Contains(body, `name="exclude" value="Ben"`) {
		t.Error("joiner offered as own exclusion")
	}
	for _, name := range []string{"Nobody", "Anna"} {
		body := c.do("GET", join+"?name="+name, "en", nil).Body.String()
		if strings.Contains(body, "Hello") || !strings.Contains(body, "Cleo") {
			t.Errorf("%s: want selection again:\n%s", name, body)
		}
	}
}

func TestJoinValidationKeepsInput(t *testing.T) {
	tests := []struct {
		name string
		in   url.Values
		want string
	}{
		{"bad email", joinForm("Ben", "nope", "wish"), "valid email"},
		{"empty wish", joinForm("Ben", "ben@x.de", " "), "enter a wish"},
		{"long wish", joinForm("Ben", "ben@x.de", strings.Repeat("x", 2001)), "at most 2000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, t.TempDir())
			_, join := openJoinSession(t, c)
			before := len(c.mailer.sent)
			rec := c.do("POST", join, "en", tt.in)
			body := rec.Body.String()
			if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, tt.want) || !strings.Contains(body, "Hello Ben!") {
				t.Errorf("status %d, want %q:\n%s", rec.Code, tt.want, body)
			}
			if tt.name != "bad email" && !strings.Contains(body, `value="ben@x.de"`) {
				t.Error("email not kept")
			}
			if len(c.mailer.sent) != before || !strings.Contains(c.do("GET", join, "en", nil).Body.String(), "Ben") {
				t.Error("failed join had effects")
			}
		})
	}
}

func TestJoinRejectsInfeasibleExclusions(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join := openJoinSession(t, c)
	// Cleo excludes everyone but Anna; Ben joining and excluding Anna leaves no Draw.
	c.do("POST", join, "en", joinForm("Cleo", "cleo@x.de", "w", "Ben"))
	rec := c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w", "Cleo", "Anna"))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "no valid Draw") {
		t.Errorf("status %d:\n%s", rec.Code, rec.Body.String())
	}
}

func TestJoinRedirectsToWorkingPersonalLinkAndConfirmsInJoinersLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join := openJoinSession(t, c)
	before := len(c.mailer.sent)
	rec := c.do("POST", join, "de", joinForm("Ben", "ben@x.de", "Socken"))
	loc := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !regexp.MustCompile(`^/[a-z2-7]{26}$`).MatchString(loc) {
		t.Fatalf("status %d, location %q", rec.Code, loc)
	}
	if page := c.do("GET", loc, "en", nil).Body.String(); !strings.Contains(page, "Socken") || !strings.Contains(page, "Ben") {
		t.Errorf("personal page:\n%s", page)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "ben@x.de" || !strings.Contains(sent[0].body, "Hallo Ben") || !strings.Contains(sent[0].body, "http://example.test"+loc) {
		t.Errorf("unexpected mails: %+v", sent)
	}
}

func TestSecondJoinOfSameNameIsConflict(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join := openJoinSession(t, c)
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w"))
	before := len(c.mailer.sent)
	rec := c.do("POST", join, "de", joinForm("Ben", "other@x.de", "w2"))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "gerade vergeben") || len(c.mailer.sent) != before {
		t.Errorf("status %d, mails %d:\n%s", rec.Code, len(c.mailer.sent)-before, rec.Body.String())
	}
}

func TestLastJoinTriggersDrawToJoinedAddressesThenJoiningIsClosed(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join, anna := createAnnaSession(t, c, "en")
	c.do("POST", anna, "en", c.personalForm(anna, "a"))
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "b"))
	before := len(c.mailer.sent)
	c.do("POST", join, "en", joinForm("Cleo", "cleo@x.de", "c"))
	draw := map[string]bool{}
	for _, m := range c.mailer.sent[before:] {
		draw[m.to] = strings.Contains(m.subject, "assignment")
	}
	if !draw["anna@x.de"] || !draw["ben@x.de"] || !draw["cleo@x.de"] {
		t.Errorf("draw mails: %v", draw)
	}
	if body := c.do("GET", join, "en", nil).Body.String(); !strings.Contains(body, "closed") || strings.Contains(body, "Cleo") {
		t.Errorf("want closed page:\n%s", body)
	}
	if rec := c.do("POST", join, "en", joinForm("Cleo", "z@x.de", "w")); rec.Code != http.StatusConflict {
		t.Errorf("join after draw: %d", rec.Code)
	}
}

func TestJoinLinkSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	_, join := openJoinSession(t, newClient(t, dir))
	rebuilt := newClient(t, dir)
	if body := rebuilt.do("GET", join, "en", nil).Body.String(); !strings.Contains(body, "Cleo") {
		t.Errorf("join link lost:\n%s", body)
	}
	rec := rebuilt.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w"))
	loc := rec.Header().Get("Location")
	if page := newClient(t, dir).do("GET", loc, "en", nil); page.Code != 200 {
		t.Errorf("personal link of join not persisted: %d", page.Code)
	}
}
