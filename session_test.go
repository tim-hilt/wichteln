package main

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

type sentMail struct{ to, subject, body string }

// fakeMailer records every email instead of sending it.
type fakeMailer struct{ sent []sentMail }

func (m *fakeMailer) Send(to, subject, body string) error {
	m.sent = append(m.sent, sentMail{to, subject, body})
	return nil
}

type client struct {
	t      *testing.T
	h      http.Handler
	mailer *fakeMailer
}

func newClient(t *testing.T, dir string) *client {
	t.Helper()
	m := &fakeMailer{}
	return &client{t, NewApp(dir, m, "http://example.test", rand.Reader), m}
}

func (c *client) do(method, path, lang string, form url.Values) *httptest.ResponseRecorder {
	c.t.Helper()
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if lang != "" {
		req.Header.Set("Accept-Language", lang)
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

// createSession drives "create" and returns the Session link path.
func (c *client) createSession() string {
	c.t.Helper()
	rec := c.do("POST", "/new", "", url.Values{})
	loc := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !regexp.MustCompile(`^/session/[a-z2-7]{26}$`).MatchString(loc) {
		c.t.Fatalf("create: status %d, location %q", rec.Code, loc)
	}
	return loc
}

func form(rows ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(rows); i += 2 {
		v.Add("name", rows[i])
		v.Add("email", rows[i+1])
	}
	return v
}

var threeRows = form("Anna", "anna@x.de", "Ben", "ben@x.de", "Cleo", "cleo@x.de")

func TestCreateRedirectsToFreshSessionLinks(t *testing.T) {
	c := newClient(t, t.TempDir())
	if a, b := c.createSession(), c.createSession(); a == b {
		t.Errorf("session links not fresh: %s", a)
	}
}

func TestUnknownSessionShowsEmptyFormAndStoresNothing(t *testing.T) {
	dir := t.TempDir()
	c := newClient(t, dir)
	rec := c.do("GET", c.createSession(), "en", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "<form") || !strings.Contains(body, "Me") {
		t.Errorf("want creation form with Me row, got %d:\n%s", rec.Code, body)
	}
	c2 := newClient(t, dir)
	if rec := c2.do("GET", "/session/"+strings.Repeat("a", 26), "en", nil); !strings.Contains(rec.Body.String(), "<form") {
		t.Error("session was stored before submit")
	}
	if rec := c.do("GET", "/session/not-a-hash", "en", nil); rec.Code != 404 {
		t.Errorf("malformed hash: status %d", rec.Code)
	}
}

func TestGermanFormLabelsOrganizerRowIch(t *testing.T) {
	c := newClient(t, t.TempDir())
	if body := c.do("GET", c.createSession(), "de", nil).Body.String(); !strings.Contains(body, "Ich") {
		t.Errorf("missing Ich:\n%s", body)
	}
}

func TestParticipantRowFragment(t *testing.T) {
	c := newClient(t, t.TempDir())
	rec := c.do("GET", "/participant-row", "en", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `name="email"`) || strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("want bare row fragment, got %d:\n%s", rec.Code, rec.Body.String())
	}
}

func TestSubmitCreatesSessionAndInvitesEveryone(t *testing.T) {
	c := newClient(t, t.TempDir())
	link := c.createSession()
	rec := c.do("POST", link, "en", threeRows)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != link {
		t.Fatalf("submit: status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
	if len(c.mailer.sent) != 3 {
		t.Fatalf("want 3 invitations, got %d", len(c.mailer.sent))
	}
	links := map[string]bool{}
	for i, to := range []string{"anna@x.de", "ben@x.de", "cleo@x.de"} {
		m := c.mailer.sent[i]
		found := regexp.MustCompile(`http://example\.test/[a-z2-7]{26}`).FindString(m.body)
		if m.to != to || found == "" || !strings.Contains(m.body, "Invitation") && !strings.Contains(m.subject, "Invitation") {
			t.Errorf("bad invitation %d: %+v", i, m)
		}
		links[found] = true
	}
	if len(links) != 3 {
		t.Errorf("Personal links not distinct: %v", links)
	}

	page := c.do("GET", link, "en", nil).Body.String()
	for _, want := range []string{"Anna", "anna@x.de", "Cleo", "pending"} {
		if !strings.Contains(page, want) {
			t.Errorf("session page missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, `name="name"`) {
		t.Error("participants still editable after submit")
	}
}

func TestInvitationsUseOrganizerLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	c.do("POST", c.createSession(), "de", threeRows)
	if m := c.mailer.sent[0]; !strings.Contains(m.subject+m.body, "Einladung") {
		t.Errorf("want German invitation: %+v", m)
	}
}

func TestSubmitToExistingSessionIsRejected(t *testing.T) {
	c := newClient(t, t.TempDir())
	link := c.createSession()
	c.do("POST", link, "en", threeRows)
	rec := c.do("POST", link, "en", form("X", "x@x.de", "Y", "y@x.de", "Z", "z@x.de"))
	if rec.Code != http.StatusConflict || len(c.mailer.sent) != 3 {
		t.Errorf("status %d, mails %d", rec.Code, len(c.mailer.sent))
	}
}

func TestValidationErrorsKeepRows(t *testing.T) {
	tests := []struct {
		name string
		in   url.Values
		want string
	}{
		{"too few", form("Anna", "anna@x.de", "Ben", "ben@x.de"), "at least 3"},
		{"empty name", form("Anna", "anna@x.de", " ", "ben@x.de", "Cleo", "cleo@x.de"), "Names must not be empty"},
		{"duplicate name", form("Anna", "anna@x.de", "ANNA", "ben@x.de", "Cleo", "cleo@x.de"), "must be unique"},
		{"bad email", form("Anna", "anna@x.de", "Ben", "ben-at-x", "Cleo", "cleo@x.de"), "valid email"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, t.TempDir())
			link := c.createSession()
			rec := c.do("POST", link, "en", tt.in)
			body := rec.Body.String()
			if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, tt.want) {
				t.Errorf("status %d, want %q in:\n%s", rec.Code, tt.want, body)
			}
			if !strings.Contains(body, `value="Anna"`) || !strings.Contains(body, `value="anna@x.de"`) {
				t.Error("entered rows not kept")
			}
			if len(c.mailer.sent) != 0 {
				t.Error("emails sent despite error")
			}
			if again := c.do("GET", link, "en", nil).Body.String(); !strings.Contains(again, "<form") || strings.Contains(again, "pending") {
				t.Error("invalid session was stored")
			}
		})
	}
}

func TestValidationErrorsAreLocalized(t *testing.T) {
	c := newClient(t, t.TempDir())
	body := c.do("POST", c.createSession(), "de", form("A", "a@x.de")).Body.String()
	if !strings.Contains(body, "mindestens 3") {
		t.Errorf("want German error:\n%s", body)
	}
}

func TestDuplicateEmailsAllowed(t *testing.T) {
	c := newClient(t, t.TempDir())
	rec := c.do("POST", c.createSession(), "en", form("Anna", "f@x.de", "Ben", "f@x.de", "Cleo", "f@x.de"))
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status %d", rec.Code)
	}
}

func TestSessionSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	c := newClient(t, dir)
	link := c.createSession()
	c.do("POST", link, "en", threeRows)

	rebuilt := newClient(t, dir)
	if page := rebuilt.do("GET", link, "en", nil).Body.String(); !strings.Contains(page, "Anna") || !strings.Contains(page, "pending") {
		t.Errorf("session lost after rebuild:\n%s", page)
	}
	if rec := rebuilt.do("POST", link, "en", threeRows); rec.Code != http.StatusConflict {
		t.Errorf("rebuilt app accepted duplicate: %d", rec.Code)
	}
}
