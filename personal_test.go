package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

var personalLinkRe = regexp.MustCompile(`/([a-z2-7]{26})\n`)

// openSession creates a Session of Anna, Ben and Cleo and returns the Session
// link plus each Participant's Personal link path (e.g. "/abc...").
func openSession(t *testing.T, c *client) (session string, personal map[string]string) {
	t.Helper()
	session = c.createSession()
	c.do("POST", session, "en", threeRows)
	personal = map[string]string{}
	for _, m := range c.mailer.sent {
		personal[m.to] = "/" + personalLinkRe.FindStringSubmatch(m.body)[1]
	}
	return session, map[string]string{
		"Anna": personal["anna@x.de"], "Ben": personal["ben@x.de"], "Cleo": personal["cleo@x.de"],
	}
}

func wishForm(wish string, excluded ...string) url.Values {
	v := url.Values{"wish": {wish}}
	for _, e := range excluded {
		v.Add("exclude", e)
	}
	return v
}

func TestPersonalLinkGreetsByNameAndOffersOtherParticipants(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	rec := c.do("GET", p["Ben"], "en", nil)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Ben") {
		t.Fatalf("status %d, body:\n%s", rec.Code, body)
	}
	for _, want := range []string{`value="Anna"`, `value="Cleo"`, "only restrict whom you give to"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, `value="Ben"`) {
		t.Error("Participant can exclude themselves")
	}
}

func TestUnknownPersonalLinkIs404InUserLanguage(t *testing.T) {
	c := newClient(t, t.TempDir())
	openSession(t, c)
	for _, path := range []string{"/" + strings.Repeat("a", 26), "/nonsense"} {
		rec := c.do("GET", path, "de", nil)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Nicht gefunden") {
			t.Errorf("%s: status %d, body %q", path, rec.Code, rec.Body.String())
		}
	}
	if rec := c.do("POST", "/"+strings.Repeat("a", 26), "en", wishForm("x")); rec.Code != http.StatusNotFound {
		t.Errorf("POST to unknown link: %d", rec.Code)
	}
}

func TestSubmitStoresWishAndExclusionsAndMarksSubmitted(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)

	rec := c.do("POST", p["Anna"], "de", wishForm("Socken", "Ben"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != p["Anna"] {
		t.Fatalf("status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}

	page := c.do("GET", p["Anna"], "en", nil).Body.String()
	if !strings.Contains(page, "Socken") {
		t.Errorf("saved wish not shown:\n%s", page)
	}
	if !regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(page) || regexp.MustCompile(`value="Cleo"[^>]*checked`).MatchString(page) {
		t.Errorf("saved exclusions not shown:\n%s", page)
	}

	status := c.do("GET", session, "en", nil).Body.String()
	if !strings.Contains(status, "submitted") {
		t.Errorf("session page lacks submitted status:\n%s", status)
	}
}

func TestEditKeepsSubmittedAndReplacesData(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Ben"))
	c.do("POST", p["Anna"], "en", wishForm("Tee", "Cleo"))

	page := c.do("GET", p["Anna"], "en", nil).Body.String()
	if !strings.Contains(page, "Tee") || strings.Contains(page, "Socken") {
		t.Errorf("wish not replaced:\n%s", page)
	}
	if regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(page) || !regexp.MustCompile(`value="Cleo"[^>]*checked`).MatchString(page) {
		t.Errorf("exclusions not replaced:\n%s", page)
	}
	if status := c.do("GET", session, "en", nil).Body.String(); strings.Count(status, "submitted") != 1 {
		t.Errorf("want exactly one submitted:\n%s", status)
	}
}

func TestWishValidation(t *testing.T) {
	tests := []struct {
		name, wish, want string
	}{
		{"empty", "", "Please enter a wish"},
		{"blank", "   \n ", "Please enter a wish"},
		{"too long", strings.Repeat("ä", 2001), "at most 2000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(t, t.TempDir())
			session, p := openSession(t, c)
			rec := c.do("POST", p["Anna"], "en", wishForm(tt.wish, "Ben"))
			if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("status %d, body:\n%s", rec.Code, rec.Body.String())
			}
			if !regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(rec.Body.String()) {
				t.Error("exclusions not kept on error")
			}
			if strings.Contains(c.do("GET", session, "en", nil).Body.String(), "submitted") {
				t.Error("invalid submission marked as submitted")
			}
		})
	}
	t.Run("2000 characters fit", func(t *testing.T) {
		c := newClient(t, t.TempDir())
		_, p := openSession(t, c)
		if rec := c.do("POST", p["Anna"], "en", wishForm(strings.Repeat("ä", 2000))); rec.Code != http.StatusSeeOther {
			t.Errorf("status %d", rec.Code)
		}
	})
}

func TestImpossibleExclusionsAreRejectedAndNothingIsSaved(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Ben"))
	c.do("POST", p["Ben"], "en", wishForm("Tee", "Cleo"))

	// Anna->Cleo, Ben->Anna are forced, so Cleo may only give to Ben.
	// Excluding Ben leaves no valid Draw.
	for lang, want := range map[string]string{"en": "no valid Draw", "de": "keine gültige Auslosung"} {
		rec := c.do("POST", p["Cleo"], lang, wishForm("Kerze", "Ben"))
		body := rec.Body.String()
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, want) {
			t.Fatalf("%s: status %d, body:\n%s", lang, rec.Code, body)
		}
		if !strings.Contains(body, "Kerze") || !regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(body) {
			t.Errorf("%s: input not kept:\n%s", lang, body)
		}
	}

	page := c.do("GET", p["Cleo"], "en", nil).Body.String()
	if strings.Contains(page, "Kerze") || strings.Contains(page, "checked") {
		t.Errorf("rejected submission was stored:\n%s", page)
	}
	if status := c.do("GET", session, "en", nil).Body.String(); strings.Count(status, "submitted") != 2 {
		t.Errorf("want exactly two submitted:\n%s", status)
	}
}

func TestValidationErrorIsLocalized(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	if body := c.do("POST", p["Anna"], "de", wishForm("")).Body.String(); !strings.Contains(body, "Bitte einen Wunsch") {
		t.Errorf("not localized:\n%s", body)
	}
}

func TestUnknownAndSelfExclusionsAreIgnored(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Anna", "Nobody"))
	if page := c.do("GET", p["Anna"], "en", nil).Body.String(); strings.Contains(page, "checked") || strings.Contains(page, "Nobody") {
		t.Errorf("bogus exclusions kept:\n%s", page)
	}
}

func TestPrivacyOfWishesAndExclusions(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Anna"], "en", wishForm("Geheimwunsch", "Ben"))

	if page := c.do("GET", p["Ben"], "en", nil).Body.String(); strings.Contains(page, "Geheimwunsch") || strings.Contains(page, "checked") {
		t.Errorf("Ben's page leaks Anna's data:\n%s", page)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "Geheimwunsch") || strings.Contains(page, "checked") {
		t.Errorf("Session page leaks data:\n%s", page)
	}
}

func TestSubmissionSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	c := newClient(t, dir)
	_, p := openSession(t, c)
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Ben"))
	if page := newClient(t, dir).do("GET", p["Anna"], "en", nil).Body.String(); !strings.Contains(page, "Socken") {
		t.Errorf("wish lost after restart:\n%s", page)
	}
}
