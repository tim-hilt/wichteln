package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

var (
	personalLinkRe = regexp.MustCompile(`/([a-z2-7]{26})\n`)
	joinLinkRe     = regexp.MustCompile(`/join/[a-z2-7]{26}`)
)

// createAnnaSession creates a Session of Anna (the Organizer, anna@x.de), Ben
// and Cleo, and returns the Session link, the Join link path and Anna's
// Personal link path, all taken from the Organizer email.
func createAnnaSession(t *testing.T, c *client, lang string) (session, join, anna string) {
	t.Helper()
	session = c.createSession()
	if rec := c.do("POST", session, lang, threeRows); rec.Code != http.StatusSeeOther {
		t.Fatalf("create: status %d:\n%s", rec.Code, rec.Body.String())
	}
	if len(c.mailer.sent) == 0 {
		t.Fatal("no Organizer email")
	}
	body := c.mailer.sent[len(c.mailer.sent)-1].body
	join = joinLinkRe.FindString(body)
	// The Personal link is the Organizer email's last link.
	if m := personalLinkRe.FindAllStringSubmatch(body, -1); join != "" && m != nil {
		return session, join, "/" + m[len(m)-1][1]
	}
	t.Fatalf("Organizer email lacks Join or Personal link:\n%s", body)
	return
}

// openJoinSession creates a Session where only Anna (the Organizer) has
// joined, and returns the Session link and the Join link path.
func openJoinSession(t *testing.T, c *client) (session, join string) {
	t.Helper()
	session, join, _ = createAnnaSession(t, c, "en")
	return session, join
}

// openSession creates a Session of Anna, Ben and Cleo where Ben and Cleo have
// joined through the Join link (wish "<name>-wish"); only Anna's Submission is
// missing. It returns the Session link plus each Participant's Personal link
// path (e.g. "/abc...").
func openSession(t *testing.T, c *client) (session string, personal map[string]string) {
	t.Helper()
	session, join, anna := createAnnaSession(t, c, "en")
	personal = map[string]string{"Anna": anna}
	for _, n := range []string{"Ben", "Cleo"} {
		rec := c.do("POST", join, "en", joinForm(n, strings.ToLower(n)+"@x.de", n+"-wish"))
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("join %s: status %d", n, rec.Code)
		}
		personal[n] = rec.Header().Get("Location")
	}
	return session, personal
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
	session, _, anna := createAnnaSession(t, c, "en")
	p := map[string]string{"Anna": anna}

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
	if !strings.Contains(status, ">joined<") {
		t.Errorf("session page lacks submitted status:\n%s", status)
	}
}

func TestEditKeepsSubmittedAndReplacesData(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _, anna := createAnnaSession(t, c, "en")
	p := map[string]string{"Anna": anna}
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Ben"))
	c.do("POST", p["Anna"], "en", wishForm("Tee", "Cleo"))

	page := c.do("GET", p["Anna"], "en", nil).Body.String()
	if !strings.Contains(page, "Tee") || strings.Contains(page, "Socken") {
		t.Errorf("wish not replaced:\n%s", page)
	}
	if regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(page) || !regexp.MustCompile(`value="Cleo"[^>]*checked`).MatchString(page) {
		t.Errorf("exclusions not replaced:\n%s", page)
	}
	if status := c.do("GET", session, "en", nil).Body.String(); strings.Count(status, ">joined<") != 1 {
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
			session, _, anna := createAnnaSession(t, c, "en")
			p := map[string]string{"Anna": anna}
			rec := c.do("POST", p["Anna"], "en", wishForm(tt.wish, "Ben"))
			if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("status %d, body:\n%s", rec.Code, rec.Body.String())
			}
			if !regexp.MustCompile(`value="Ben"[^>]*checked`).MatchString(rec.Body.String()) {
				t.Error("exclusions not kept on error")
			}
			if strings.Contains(c.do("GET", session, "en", nil).Body.String(), ">joined<") {
				t.Error("invalid submission marked as submitted")
			}
		})
	}
	t.Run("2000 characters fit", func(t *testing.T) {
		c := newClient(t, t.TempDir())
		_, _, anna := createAnnaSession(t, c, "en")
		p := map[string]string{"Anna": anna}
		if rec := c.do("POST", p["Anna"], "en", wishForm(strings.Repeat("ä", 2000))); rec.Code != http.StatusSeeOther {
			t.Errorf("status %d", rec.Code)
		}
	})
}

func TestImpossibleExclusionsAreRejectedAndNothingIsSaved(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Ben"], "en", wishForm("Tee", "Cleo"))
	c.do("POST", p["Cleo"], "en", wishForm("Kerze", "Anna"))

	// Ben->Anna and Cleo->Ben are forced, so Anna may only give to Cleo.
	// Excluding Cleo leaves no valid Draw.
	for lang, want := range map[string]string{"en": "no valid Draw", "de": "keine gültige Auslosung"} {
		rec := c.do("POST", p["Anna"], lang, wishForm("Socken", "Cleo"))
		body := rec.Body.String()
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(body, want) {
			t.Fatalf("%s: status %d, body:\n%s", lang, rec.Code, body)
		}
		if !strings.Contains(body, "Socken") || !regexp.MustCompile(`value="Cleo"[^>]*checked`).MatchString(body) {
			t.Errorf("%s: input not kept:\n%s", lang, body)
		}
	}

	page := c.do("GET", p["Anna"], "en", nil).Body.String()
	if strings.Contains(page, "Socken") || strings.Contains(page, "checked") {
		t.Errorf("rejected submission was stored:\n%s", page)
	}
	if status := c.do("GET", session, "en", nil).Body.String(); strings.Count(status, ">joined<") != 2 {
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
	_, _, anna := createAnnaSession(t, c, "en")
	p := map[string]string{"Anna": anna}
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Anna", "Nobody"))
	if page := c.do("GET", p["Anna"], "en", nil).Body.String(); strings.Contains(page, "checked") || strings.Contains(page, "Nobody") {
		t.Errorf("bogus exclusions kept:\n%s", page)
	}
}

func TestPrivacyOfWishesAndExclusions(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "en") // Cleo stays unjoined: no Draw
	ben := c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "b")).Header().Get("Location")
	p := map[string]string{"Anna": anna, "Ben": ben}
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
	_, _, anna := createAnnaSession(t, c, "en")
	p := map[string]string{"Anna": anna}
	c.do("POST", p["Anna"], "en", wishForm("Socken", "Ben"))
	if page := newClient(t, dir).do("GET", p["Anna"], "en", nil).Body.String(); !strings.Contains(page, "Socken") {
		t.Errorf("wish lost after restart:\n%s", page)
	}
}
