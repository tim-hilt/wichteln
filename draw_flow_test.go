package main

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// submitAll submits Ben, Cleo and Anna (the last missing Submission, see
// openSession) with wish "<name>-wish" and the given exclusions, and returns
// the mails sent by the Draw.
func submitAll(t *testing.T, c *client, p map[string]string, lang string, excl map[string][]string) []sentMail {
	t.Helper()
	before := len(c.mailer.sent)
	for _, n := range []string{"Ben", "Cleo", "Anna"} {
		if rec := c.do("POST", p[n], lang, c.personalForm(p[n], n+"-wish", excl[n]...)); rec.Code != http.StatusSeeOther {
			t.Fatalf("%s: status %d", n, rec.Code)
		}
	}
	return c.mailer.sent[before:]
}

func TestNoDrawUntilLastSubmission(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	before := len(c.mailer.sent) // Ben and Cleo have joined, Anna has not submitted
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "b"))
	if len(c.mailer.sent) != before {
		t.Errorf("mail sent before last Submission")
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "names have been drawn") {
		t.Errorf("drawn too early:\n%s", page)
	}
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a"))
	if page := c.do("GET", session, "en", nil).Body.String(); !strings.Contains(page, "names have been drawn") {
		t.Errorf("not drawn after last Submission:\n%s", page)
	}
}

func TestFullFlowDrawSendsValidAssignmentsToEveryone(t *testing.T) {
	emails := map[string]string{"Anna": "anna@x.de", "Ben": "ben@x.de", "Cleo": "cleo@x.de"}
	for range 20 {
		c := newClient(t, t.TempDir())
		session, p := openSession(t, c)
		excl := map[string][]string{"Anna": {"Ben"}, "Ben": {"Cleo"}}
		mails := submitAll(t, c, p, "en", excl)
		if len(mails) != 3 {
			t.Fatalf("want 3 draw mails, got %d", len(mails))
		}
		gives, receives := map[string]string{}, map[string]int{}
		for _, m := range mails {
			giver := ""
			for n, e := range emails {
				if e == m.to {
					giver = n
				}
			}
			recipient := ""
			for n := range emails {
				if strings.Contains(m.body, n+"-wish") {
					recipient = n
				}
			}
			if giver == "" || recipient == "" || !strings.Contains(m.body, recipient) || !strings.Contains(m.body, "Hello "+giver) {
				t.Fatalf("bad mail to %s:\n%s", m.to, m.body)
			}
			gives[giver] = recipient
			receives[recipient]++
		}
		for n := range emails {
			if gives[n] == "" || gives[n] == n || receives[n] != 1 {
				t.Fatalf("invalid assignments %v", gives)
			}
			for _, x := range excl[n] {
				if gives[n] == x {
					t.Fatalf("exclusion violated: %v", gives)
				}
			}
			// Personal link shows the same recipient and wish, read-only.
			page := c.do("GET", p[n], "en", nil).Body.String()
			if !strings.Contains(page, gives[n]+"-wish") || strings.Contains(page, "<form") || strings.Contains(page, "<textarea") {
				t.Fatalf("post-draw page of %s:\n%s", n, page)
			}
			for other := range emails {
				if other != gives[n] && other != n && strings.Contains(page, other+"-wish") {
					t.Fatalf("page of %s leaks %s's wish", n, other)
				}
			}
			if strings.Contains(page, "gives to you") {
				t.Fatalf("page reveals giver")
			}
		}
		status := c.do("GET", session, "en", nil).Body.String()
		if !strings.Contains(status, "The names have been drawn.") || strings.Contains(status, "-wish") || strings.Contains(status, "checked") {
			t.Fatalf("session page after draw:\n%s", status)
		}
	}
}

func TestSimultaneousLastSubmissionsDrawOnce(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Go(func() { codes[i] = c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a")).Code })
	}
	wg.Wait()
	if drawMails := len(c.mailer.sent) - before; drawMails != 3 {
		t.Errorf("want 3 draw mails, got %d (statuses %v)", drawMails, codes)
	}
}

func TestFailedDrawEmailDoesNotStopTheOthers(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	c.mailer.failTo = "anna@x.de" // the first Participant
	if rec := c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a")); rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d", rec.Code)
	}
	if got := len(c.mailer.sent) - before; got != 2 {
		t.Errorf("want draw mails to Ben and Cleo, got %d", got)
	}
}

func TestDrawEmailUsesLanguageOfLastSubmission(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	c.do("POST", p["Cleo"], "de", c.personalForm(p["Cleo"], "Cleo-wish")) // edit in German
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "Anna-wish")) // last Submission
	for _, m := range c.mailer.sent[before:] {
		german := strings.Contains(m.subject, "Zuteilung")
		if (m.to == "cleo@x.de") != german {
			t.Errorf("mail to %s subject %q", m.to, m.subject)
		}
	}
}

func TestPostDrawSubmissionsAreRejected(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	submitAll(t, c, p, "en", nil)
	before := len(c.mailer.sent)
	form := wishForm("changed")
	form.Set("email", "anna@new.de")
	rec := c.do("POST", p["Anna"], "en", form)
	if rec.Code != http.StatusConflict {
		t.Errorf("status %d", rec.Code)
	}
	if strings.Contains(c.do("GET", p["Anna"], "en", nil).Body.String(), "changed") || len(c.mailer.sent) != before {
		t.Error("post-draw edit had an effect")
	}
}

func TestDrawEmailHTMLShowsEscapedWish(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	for _, n := range []string{"Ben", "Cleo", "Anna"} {
		c.do("POST", p[n], "en", c.personalForm(p[n], "Books & <cookies>"))
	}
	mails := c.mailer.sent[before:]
	if len(mails) != 3 {
		t.Fatalf("want 3 draw mails, got %d", len(mails))
	}
	for _, m := range mails {
		if !strings.Contains(m.html, "Books &amp; &lt;cookies&gt;") || strings.Contains(m.html, "<cookies>") || !strings.Contains(m.html, "Their wish") {
			t.Errorf("bad html draw mail to %s:\n%s", m.to, m.html)
		}
	}
}

func TestFailedConfirmationEmailDoesNotStopDrawEmails(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join, anna := createAnnaSession(t, c, "en")
	c.do("POST", anna, "en", c.personalForm(anna, "a"))
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "b"))
	before := len(c.mailer.sent)
	c.mailer.failTo = "cleo@x.de"
	if rec := c.do("POST", join, "en", joinForm("Cleo", "cleo@x.de", "c")); rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d", rec.Code)
	}
	if got := len(c.mailer.sent) - before; got != 2 {
		t.Errorf("want draw mails to Anna and Ben, got %d", got)
	}
}

func TestFailedChangedEmailConfirmationDoesNotStopDrawEmails(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	before := len(c.mailer.sent)
	c.mailer.failTo = "anna@new.de"
	form := c.personalForm(p["Anna"], "a")
	form.Set("email", "anna@new.de")
	if rec := c.do("POST", p["Anna"], "en", form); rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d", rec.Code)
	}
	if got := len(c.mailer.sent) - before; got != 2 {
		t.Errorf("want draw mails to Ben and Cleo, got %d", got)
	}
}
