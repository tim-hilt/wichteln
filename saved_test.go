package main

import (
	"net/http"
	"strings"
	"testing"
)

// clearsQuery is the script on a page with a saved banner that removes the
// query from the URL, so a reload doesn't show the banner again.
const clearsQuery = `history.replaceState(null, "", location.pathname)`

// assertSavedRedirect checks that a successful save redirects to path with
// the given query, that the page there shows the banner text want and clears
// the query, and that path without the query shows no banner.
func assertSavedRedirect(t *testing.T, c *client, rec *http.Response, path, query, want string) {
	t.Helper()
	if loc := rec.Header.Get("Location"); rec.StatusCode != http.StatusSeeOther || loc != path+query {
		t.Fatalf("status %d, location %q, want %q", rec.StatusCode, loc, path+query)
	}
	page := c.do("GET", path+query, "en", nil).Body.String()
	if !strings.Contains(page, `role="status"`) || !strings.Contains(page, want) || !strings.Contains(page, clearsQuery) {
		t.Errorf("no saved banner %q:\n%s", want, page)
	}
	if page := c.do("GET", path, "en", nil).Body.String(); strings.Contains(page, `role="status"`) {
		t.Errorf("banner without ?saved:\n%s", page)
	}
}

func TestPersonalSaveShowsSavedBanner(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	rec := c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "new wish"))
	assertSavedRedirect(t, c, rec.Result(), p["Ben"], "?saved", "Your changes have been saved.")
}

func TestPersonalSaveWithNewEmailMentionsLinksEmail(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, p := openSession(t, c)
	form := c.personalForm(p["Ben"], "w")
	form.Set("email", "ben@new.de")
	rec := c.do("POST", p["Ben"], "en", form)
	assertSavedRedirect(t, c, rec.Result(), p["Ben"], "?saved=email", "We sent your links to ben@new.de.")
}

func TestSessionSaveShowsSavedBanner(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	// Saving the names unchanged emails nobody.
	rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo")
	assertSavedRedirect(t, c, rec.Result(), session, "?saved", "Your changes have been saved.")
}

func TestSessionSaveThatEmailsMentionsNotification(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	assertSavedRedirect(t, c, rec.Result(), session, "?saved=notified", "Joined participants were notified by email.")
}

func TestSavedBannerIsGerman(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	if page := c.do("GET", session+"?saved", "de", nil).Body.String(); !strings.Contains(page, "Deine Änderungen wurden gespeichert.") {
		t.Errorf("no German banner:\n%s", page)
	}
}
