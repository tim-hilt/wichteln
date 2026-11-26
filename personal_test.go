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
		anna = "/" + m[len(m)-1][1]
		c.do("GET", anna, lang, nil) // dismiss the confirmation
		return session, join, anna
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
		c.do("GET", personal[n], "en", nil)
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

// personalForm is the form a browser would send from the Personal link page
// at path: the page's prefilled email plus the given wish and exclusions.
func (c *client) personalForm(path, wish string, excluded ...string) url.Values {
	c.t.Helper()
	m := regexp.MustCompile(`name="email" value="([^"]*)"`).FindStringSubmatch(c.do("GET", path, "en", nil).Body.String())
	if m == nil {
		c.t.Fatalf("no email field on %s", path)
	}
	v := wishForm(wish, excluded...)
	v.Set("email", m[1])
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
	for _, want := range []string{`value="Anna"`, `value="Cleo"`} {
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
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Dieser Link ist ungültig") {
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

	rec := c.do("POST", p["Anna"], "de", c.personalForm(p["Anna"], "Socken", "Ben"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != p["Anna"] {
		t.Fatalf("status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}

	c.do("GET", p["Anna"], "en", nil) // dismiss the Join confirmation
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
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Socken", "Ben"))
	c.do("GET", p["Anna"], "en", nil) // dismiss the Join confirmation
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Tee", "Cleo"))

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
			rec := c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], tt.wish, "Ben"))
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
		if rec := c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], strings.Repeat("ä", 2000))); rec.Code != http.StatusSeeOther {
			t.Errorf("status %d", rec.Code)
		}
	})
}

func TestImpossibleExclusionsAreRejectedAndNothingIsSaved(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "Tee", "Cleo"))
	c.do("POST", p["Cleo"], "en", c.personalForm(p["Cleo"], "Kerze", "Anna"))

	// Ben->Anna and Cleo->Ben are forced, so Anna may only give to Cleo.
	// Excluding Cleo leaves no valid Draw.
	for lang, want := range map[string]string{"en": "no draw is possible", "de": "keine Auslosung möglich"} {
		rec := c.do("POST", p["Anna"], lang, c.personalForm(p["Anna"], "Socken", "Cleo"))
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
	if body := c.do("POST", p["Anna"], "de", c.personalForm(p["Anna"], "")).Body.String(); !strings.Contains(body, "Bitte einen Wunsch") {
		t.Errorf("not localized:\n%s", body)
	}
}

func TestUnknownAndSelfExclusionsAreIgnored(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, _, anna := createAnnaSession(t, c, "en")
	p := map[string]string{"Anna": anna}
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Socken", "Anna", "Nobody"))
	if page := c.do("GET", p["Anna"], "en", nil).Body.String(); strings.Contains(page, "checked") || strings.Contains(page, "Nobody") {
		t.Errorf("bogus exclusions kept:\n%s", page)
	}
}

func TestPrivacyOfWishesAndExclusions(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "en") // Cleo stays unjoined: no Draw
	ben := c.joinAs(join, "en", joinForm("Ben", "ben@x.de", "b"))
	p := map[string]string{"Anna": anna, "Ben": ben}
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Geheimwunsch", "Ben"))

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
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Socken", "Ben"))
	c.do("GET", p["Anna"], "en", nil) // dismiss the Join confirmation
	if page := newClient(t, dir).do("GET", p["Anna"], "en", nil).Body.String(); !strings.Contains(page, "Socken") {
		t.Errorf("wish lost after restart:\n%s", page)
	}
}

func TestPersonalFormShowsCurrentEmailAsRequiredField(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	body := c.do("GET", p["Ben"], "en", nil).Body.String()
	if !strings.Contains(body, `name="email" value="ben@x.de" required`) {
		t.Errorf("email field missing:\n%s", body)
	}
}

func TestChangedEmailSendsConfirmationToNewAddressOnly(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	form := c.personalForm(p["Ben"], "b")
	form.Set("email", " ben@new.de ")
	if rec := c.do("POST", p["Ben"], "de", form); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "ben@new.de" || !strings.Contains(sent[0].body, "http://example.test"+p["Ben"]) {
		t.Fatalf("unexpected mails: %+v", sent)
	}
	if !strings.Contains(c.do("GET", p["Ben"], "en", nil).Body.String(), `value="ben@new.de"`) {
		t.Error("new email not stored")
	}
}

func TestChangedOrganizerEmailSendsCompleteOrganizerEmail(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, anna := createAnnaSession(t, c, "en")
	before := len(c.mailer.sent)
	form := c.personalForm(anna, "a")
	form.Set("email", "anna@new.de")
	c.do("POST", anna, "en", form)
	sent := c.mailer.sent[before:]
	if len(sent) != 1 || sent[0].to != "anna@new.de" {
		t.Fatalf("unexpected mails: %+v", sent)
	}
	for _, want := range []string{session, join, anna} {
		if !strings.Contains(sent[0].body, want) {
			t.Errorf("Organizer email lacks %q:\n%s", want, sent[0].body)
		}
	}
}

func TestUnchangedEmailSendsNoMail(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "new wish"))
	if n := len(c.mailer.sent) - before; n != 0 {
		t.Errorf("%d mails sent", n)
	}
}

func TestImplausibleEmailIsRejectedAndNothingIsSaved(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	for lang, want := range map[string]string{"en": "valid email", "de": "gültige E-Mail"} {
		form := c.personalForm(p["Ben"], "changed")
		form.Set("email", "nonsense")
		rec := c.do("POST", p["Ben"], lang, form)
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("%s: status %d:\n%s", lang, rec.Code, rec.Body.String())
		}
	}
	page := c.do("GET", p["Ben"], "en", nil).Body.String()
	if !strings.Contains(page, "ben@x.de") || strings.Contains(page, "changed") || len(c.mailer.sent) != before {
		t.Errorf("rejected edit had effects:\n%s", page)
	}
}

func TestOrganizerSeesJoinConfirmationAfterFirstSubmission(t *testing.T) {
	c := newClient(t, t.TempDir())
	session := c.createSession()
	c.do("POST", session, "en", threeRows)
	m := personalLinkRe.FindAllStringSubmatch(c.mailer.sent[0].body, -1)
	anna := "/" + m[len(m)-1][1]
	if page := c.do("GET", anna, "en", nil).Body.String(); strings.Contains(page, "You&#39;re in") || !strings.Contains(page, "<form") {
		t.Errorf("first visit should show the form:\n%s", page)
	}
	rec := c.do("POST", anna, "en", url.Values{"email": {"anna@x.de"}, "wish": {"w"}})
	if page := c.do("GET", rec.Header().Get("Location"), "en", nil).Body.String(); !strings.Contains(page, "You&#39;re in, Anna!") || !strings.Contains(page, "You added your wish.") || strings.Contains(page, "confirmation email") {
		t.Errorf("no organizer confirmation after submission:\n%s", page)
	}
	if page := c.do("GET", anna, "en", nil).Body.String(); !strings.Contains(page, "<form") {
		t.Errorf("later visit lacks form:\n%s", page)
	}
}
