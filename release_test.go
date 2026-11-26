package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func release(c *client, session, name string) *httptest.ResponseRecorder {
	return c.do("POST", session+"/release", "en", url.Values{"participant": {name}})
}

func TestReleaseFreesNameAndInvalidatesPersonalLink(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	ben := strings.TrimSuffix(c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w")).Header().Get("Location"), "?joined")
	before := len(c.mailer.sent)

	if rec := release(c, session, "Ben"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	if len(c.mailer.sent) != before {
		t.Error("Release sent mail")
	}
	if rec := c.do("GET", ben, "en", nil); rec.Code != http.StatusNotFound {
		t.Errorf("old Personal link: status %d", rec.Code)
	}
	if page := c.do("GET", join, "en", nil).Body.String(); !strings.Contains(page, "Ben") {
		t.Errorf("Ben not on Join page:\n%s", page)
	}
	if rec := c.do("POST", join, "en", joinForm("Ben", "ben2@x.de", "again")); rec.Code != http.StatusSeeOther {
		t.Errorf("rejoin: status %d", rec.Code)
	}
}

func TestReleaseOffersButtonOnlyForJoinedNonOrganizers(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c) // Anna joined, Ben and Cleo not
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "/release") {
		t.Errorf("Release offered:\n%s", page)
	}
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "w"))
	page := c.do("GET", session, "de", nil).Body.String()
	if n := strings.Count(page, session+"/release"); n != 1 || !strings.Contains(page, "Freigeben") {
		t.Errorf("want one Release button for Ben, got %d:\n%s", n, page)
	}
}

func TestReleaseRejectsOrganizerAndNotJoinedNames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	for _, name := range []string{"Anna", "Cleo", "Zed"} {
		if rec := release(c, session, name); rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", name, rec.Code)
		}
	}
	if rec := c.do("GET", session, "en", nil); !strings.Contains(rec.Body.String(), "Resend link") {
		t.Error("Anna was released")
	}
}

func TestReleaseAfterDrawIsConflictAndChangesNothing(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	submitAll(t, c, p, "en", nil)
	if rec := release(c, session, "Ben"); rec.Code != http.StatusConflict {
		t.Errorf("status %d", rec.Code)
	}
	if rec := c.do("GET", p["Ben"], "en", nil); rec.Code != http.StatusOK {
		t.Errorf("Ben's link: status %d", rec.Code)
	}
}

func TestReleaseOfSubmittedParticipantNeverTriggersDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	c.do("POST", p["Ben"], "en", c.personalForm(p["Ben"], "b", "Cleo"))
	before := len(c.mailer.sent)
	release(c, session, "Cleo") // the others are all submitted except Anna
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a"))
	if len(c.mailer.sent) != before {
		t.Error("mail sent")
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "Drawn ✓") {
		t.Error("drawn")
	}
}
