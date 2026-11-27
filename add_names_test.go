package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// saveNames posts the complete list of names to the Session page's edit form.
func saveNames(c *client, session, lang string, names ...string) *httptest.ResponseRecorder {
	v := url.Values{}
	for _, n := range names {
		v.Add("name", n)
	}
	return c.do("POST", session+"/names", lang, v)
}

// order returns the names in the order the Session page lists them.
func order(page string, names ...string) []string {
	re := regexp.MustCompile(`name="name" value="(` + strings.Join(names, "|") + `)"`)
	var got []string
	for _, m := range re.FindAllStringSubmatch(page, -1) {
		got = append(got, m[1])
	}
	return got
}

func TestSessionPageOffersEditFormWithFixedNames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	page := c.do("GET", session, "en", nil).Body.String()
	for _, n := range []string{"Anna", "Ben", "Cleo"} {
		if !strings.Contains(page, `type="hidden" name="name" value="`+n+`"`) {
			t.Errorf("%s not fixed in the form:\n%s", n, page)
		}
	}
	if strings.Contains(page, `<input name="name"`) || strings.Contains(page, `aria-label="Remove"`) {
		t.Errorf("a name is editable or removable:\n%s", page)
	}
	if !strings.Contains(page, `hx-get="/participant-row"`) || !strings.Contains(page, `action="`+session+`/names"`) {
		t.Errorf("no add button or save form:\n%s", page)
	}
}

func TestSaveAppendsNewNamesAfterExistingOnes(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	if rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo", " Dora ", "Emil"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d:\n%s", rec.Code, rec.Body.String())
	}
	got := order(c.do("GET", session, "en", nil).Body.String(), "Anna", "Ben", "Cleo", "Dora", "Emil")
	if strings.Join(got, ",") != "Anna,Ben,Cleo,Dora,Emil" {
		t.Errorf("order %v", got)
	}
	if page := c.do("GET", join, "en", nil).Body.String(); !strings.Contains(page, "Dora") || !strings.Contains(page, "Emil") {
		t.Errorf("new names not on Join page:\n%s", page)
	}
	if rec := c.do("POST", join, "en", joinForm("Dora", "dora@x.de", "w")); rec.Code != http.StatusSeeOther {
		t.Errorf("join added name: status %d", rec.Code)
	}
}

func TestSaveValidationErrorsKeepRowsAndChangeNothing(t *testing.T) {
	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"empty", []string{"Anna", "Ben", "Cleo", "  "}, "Names must not be empty"},
		{"duplicate", []string{"Anna", "Ben", "Cleo", "BEN"}, "Each name must be different"},
		{"duplicate of new", []string{"Anna", "Ben", "Cleo", "Dora", "dora"}, "Each name must be different"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, t.TempDir())
			session, join := openJoinSession(t, c)
			rec := saveNames(c, session, "en", tt.names...)
			body := rec.Body.String()
			if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, tt.want) {
				t.Fatalf("status %d, want %q in:\n%s", rec.Code, tt.want, body)
			}
			if !strings.Contains(body, `value="Ben"`) || !strings.Contains(body, `value="`+strings.TrimSpace(tt.names[len(tt.names)-1])+`"`) {
				t.Errorf("entered rows not kept:\n%s", body)
			}
			if page := c.do("GET", join, "en", nil).Body.String(); strings.Contains(page, "Dora") || strings.Contains(page, "BEN") {
				t.Errorf("invalid names saved:\n%s", page)
			}
		})
	}
}

func TestSaveAfterDrawIsConflictAndChangesNothing(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	submitAll(t, c, p, "en", nil)
	before := len(c.mailer.sent)
	if rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora"); rec.Code != http.StatusConflict {
		t.Errorf("status %d", rec.Code)
	}
	if strings.Contains(c.do("GET", p["Ben"], "en", nil).Body.String(), "Dora") || len(c.mailer.sent) != before {
		t.Error("Save after the Draw changed something")
	}
}

func TestSaveToUnknownSessionIsNotFound(t *testing.T) {
	c := newClient(t, t.TempDir())
	if rec := saveNames(c, c.createSession(), "en", "A", "B", "C"); rec.Code != http.StatusNotFound {
		t.Errorf("status %d", rec.Code)
	}
}

func TestAddEmailsEveryJoinedParticipantOnceInTheirLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "de") // Anna has not submitted yet
	ben := c.joinAs(join, "en", joinForm("Ben", "ben@x.de", "w"))
	c.do("POST", join, "de", joinForm("Cleo", "cleo@x.de", "w"))
	before := len(c.mailer.sent)

	if rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	mails := map[string]sentMail{}
	for _, m := range c.mailer.sent[before:] {
		if _, dup := mails[m.to]; dup {
			t.Errorf("two emails to %s", m.to)
		}
		mails[m.to] = m
	}
	if len(mails) != 3 {
		t.Fatalf("want 3 emails, got %d: %+v", len(mails), mails)
	}
	for to, want := range map[string][]string{
		"ben@x.de":  {"Hello Ben", "Dora", "http://example.test" + ben},
		"cleo@x.de": {"Hallo Cleo", "Dora", "http://example.test/"},
		"anna@x.de": {"Hallo Anna", "Dora", "http://example.test" + anna},
	} {
		for _, w := range want {
			if m := mails[to]; !strings.Contains(m.body, w) || !strings.Contains(m.html, w) {
				t.Errorf("email to %s lacks %q:\n%s", to, w, m.body)
			}
		}
	}
}

func TestAddKeepsSubmissionsValidAndOffersNewNameAsExclusion(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	if page := c.do("GET", p["Ben"], "en", nil).Body.String(); !strings.Contains(page, `name="exclude" value="Dora"`) {
		t.Errorf("Dora not in Exclusions form:\n%s", page)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Count(page, "joined") < 3 || strings.Contains(page, "pending") && strings.Count(page, "pending") > 1 {
		t.Errorf("a Submission was withdrawn:\n%s", page)
	}
}

func TestAddNeverStartsTheDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c) // only Anna's Submission is missing
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a"))
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "The names have been drawn.") {
		t.Error("drawn although Dora has not joined")
	}
}

func TestSessionEditFormIsGerman(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	page := c.do("GET", session, "de", nil).Body.String()
	if !strings.Contains(page, "Teilnehmer hinzufügen") || !strings.Contains(page, "Speichern") {
		t.Errorf("form not German:\n%s", page)
	}
}
