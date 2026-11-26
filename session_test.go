package main

import (
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
)

type sentMail struct{ to, subject, body, html string }

// fakeMailer records every email instead of sending it. Sending to failTo
// fails and records nothing. Safe for concurrent use.
type fakeMailer struct {
	mu     sync.Mutex
	sent   []sentMail
	failTo string
}

func (m *fakeMailer) Send(mail Mail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mail.To == m.failTo {
		return errors.New("mail failed")
	}
	m.sent = append(m.sent, sentMail{mail.To, mail.Subject, mail.Text, mail.HTML})
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

// form is the creation form: the Organizer's email, then one row per name,
// the first being the Organizer's.
func form(organizerEmail string, names ...string) url.Values {
	v := url.Values{"email": {organizerEmail}}
	for _, n := range names {
		v.Add("name", n)
	}
	return v
}

var threeRows = form("anna@x.de", "Anna", "Ben", "Cleo")

func TestCreateRedirectsToFreshSessionLinks(t *testing.T) {
	c := newClient(t, t.TempDir())
	if a, b := c.createSession(), c.createSession(); a == b {
		t.Errorf("session links not fresh: %s", a)
	}
}

func TestUnknownSessionShowsEmptyFormAndStoresNothing(t *testing.T) {
	dir := t.TempDir()
	c := newClient(t, dir)
	link := c.createSession()
	rec := c.do("GET", link, "en", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "<form") || !strings.Contains(body, "Me") {
		t.Errorf("want creation form with Me row, got %d:\n%s", rec.Code, body)
	}
	if rec := newClient(t, dir).do("GET", link, "en", nil); !strings.Contains(rec.Body.String(), "<form") {
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

func TestParticipantRowFragmentIsNameOnly(t *testing.T) {
	c := newClient(t, t.TempDir())
	rec := c.do("GET", "/participant-row", "en", nil)
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, `name="name"`) || strings.Contains(body, `name="email"`) || strings.Contains(body, "<html") {
		t.Errorf("want bare name-only row fragment, got %d:\n%s", rec.Code, body)
	}
}

func TestCreationFormAsksEmailOnlyInOrganizerRow(t *testing.T) {
	c := newClient(t, t.TempDir())
	body := c.do("GET", c.createSession(), "en", nil).Body.String()
	if n := strings.Count(body, `name="email"`); n != 1 {
		t.Errorf("want exactly one email field, got %d:\n%s", n, body)
	}
}

func TestCreateEmailsOnlyOrganizerWithAllThreeLinks(t *testing.T) {
	c := newClient(t, t.TempDir())
	link, join, anna := createAnnaSession(t, c, "en")
	if len(c.mailer.sent) != 1 {
		t.Fatalf("want exactly 1 email, got %d", len(c.mailer.sent))
	}
	m := c.mailer.sent[0]
	if m.to != "anna@x.de" || !strings.Contains(m.body, "Hello Anna") {
		t.Errorf("bad email: %+v", m)
	}
	for _, want := range []string{"http://example.test" + link, "http://example.test" + join, "http://example.test" + anna} {
		if !strings.Contains(m.body, want) || !strings.Contains(m.html, want) {
			t.Errorf("email missing %q:\n%s", want, m.body)
		}
	}
	if rec := c.do("GET", anna, "en", nil); rec.Code != 200 {
		t.Errorf("Personal link from email: status %d", rec.Code)
	}
}

func TestOrganizerEmailUsesOrganizerLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	createAnnaSession(t, c, "de")
	if m := c.mailer.sent[0]; !strings.Contains(m.body, "Hallo Anna") || !strings.Contains(m.body, "Beitrittslink") {
		t.Errorf("want German Organizer email: %+v", m)
	}
}

func TestSessionPageShowsStatusesAndNoEmailAddresses(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w"))
	page := c.do("GET", session, "en", nil).Body.String()
	for _, want := range []string{"Anna", "pending", "Ben", "joined", "Cleo", "not joined"} {
		if !strings.Contains(page, want) {
			t.Errorf("session page missing %q:\n%s", want, page)
		}
	}
	if strings.Contains(page, "@x.de") || strings.Contains(page, `name="email"`) || strings.Contains(page, `name="name"`) {
		t.Errorf("session page shows or asks emails, or is editable:\n%s", page)
	}
}

func TestOrganizerRowTurnsJoinedAfterTheirWish(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "en")
	row := func() string {
		page := c.do("GET", session, "en", nil).Body.String()
		return page[strings.Index(page, "Anna"):strings.Index(page, "Ben")]
	}
	if r := row(); !strings.Contains(r, "pending") {
		t.Errorf("want pending:\n%s", r)
	}
	c.do("POST", anna, "en", wishForm("w"))
	if r := row(); !strings.Contains(r, "joined") || strings.Contains(r, "pending") {
		t.Errorf("want joined:\n%s", r)
	}
	if strings.Contains(c.do("GET", join, "en", nil).Body.String(), "Anna") {
		t.Error("Organizer listed on Join page")
	}
}

func TestSubmitToExistingSessionIsRejected(t *testing.T) {
	c := newClient(t, t.TempDir())
	link := c.createSession()
	c.do("POST", link, "en", threeRows)
	rec := c.do("POST", link, "en", form("x@x.de", "X", "Y", "Z"))
	if rec.Code != http.StatusConflict || len(c.mailer.sent) != 1 {
		t.Errorf("status %d, mails %d", rec.Code, len(c.mailer.sent))
	}
}

func TestValidationErrorsKeepRows(t *testing.T) {
	tests := []struct {
		name string
		in   url.Values
		want string
	}{
		{"too few", form("anna@x.de", "Anna", "Ben"), "at least 3"},
		{"empty name", form("anna@x.de", "Anna", " ", "Cleo"), "Names must not be empty"},
		{"duplicate name", form("anna@x.de", "Anna", "ANNA", "Cleo"), "must be unique"},
		{"bad email", form("anna-at-x", "Anna", "Ben", "Cleo"), "valid email"},
		{"no email", form("", "Anna", "Ben", "Cleo"), "valid email"},
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
			if !strings.Contains(body, `value="Anna"`) || !strings.Contains(body, `value="`+tt.in.Get("email")+`"`) {
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
	body := c.do("POST", c.createSession(), "de", form("a@x.de", "A")).Body.String()
	if !strings.Contains(body, "mindestens 3") {
		t.Errorf("want German error:\n%s", body)
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
