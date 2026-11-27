package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestSaveRemovesUnjoinedNamesAndRenames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c) // Anna joined; Ben, Cleo not
	// Swapping one name at exactly 3 Participants works: remove plus add.
	if rec := saveNames(c, session, "en", "Anna", "Ben", "Clea"); rec.Code != http.StatusSeeOther {
		t.Fatalf("rename: status %d:\n%s", rec.Code, rec.Body.String())
	}
	page := c.do("GET", join, "en", nil).Body.String()
	if strings.Contains(page, "Cleo") || !strings.Contains(page, "Clea") {
		t.Errorf("rename not applied:\n%s", page)
	}
	// Removal and addition in one Save.
	if rec := saveNames(c, session, "en", "Anna", "Clea", "Dora", "Emil"); rec.Code != http.StatusSeeOther {
		t.Fatalf("remove+add: status %d", rec.Code)
	}
	got := order(c.do("GET", session, "en", nil).Body.String(), "Anna", "Ben", "Clea", "Dora", "Emil")
	if strings.Join(got, ",") != "Anna,Clea,Dora,Emil" {
		t.Errorf("order %v", got)
	}
}

func TestSaveOffersUnjoinedNamesRemovableButNotEditable(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openJoinSession(t, c)
	page := c.do("GET", session, "en", nil).Body.String()
	if !strings.Contains(page, `type="hidden" name="name" value="Ben"`) || strings.Contains(page, `<input name="name"`) || !strings.Contains(page, `aria-label="Remove"`) {
		t.Errorf("Ben editable or not removable:\n%s", page)
	}
	if !strings.Contains(page, `type="hidden" name="name" value="Anna"`) {
		t.Errorf("Anna not fixed:\n%s", page)
	}
}

func TestSaveBelowThreeParticipantsIsRejected(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	rec := saveNames(c, session, "en", "Anna", "Ben")
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "at least 3 participants") {
		t.Errorf("status %d:\n%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(c.do("GET", join, "en", nil).Body.String(), "Cleo") {
		t.Error("rejected Save removed Cleo")
	}
}

func TestSaveCannotRemoveJoinedNamesOrTheOrganizer(t *testing.T) {
	for name, names := range map[string][]string{
		"joined":    {"Anna", "Ben", "Dora"},
		"organizer": {"Ben", "Cleo", "Dora"},
	} {
		t.Run(name, func(t *testing.T) {
			c := newClient(t, t.TempDir())
			session, p := openSession(t, c) // all three joined
			rec := saveNames(c, session, "en", names...)
			if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "joined") {
				t.Errorf("status %d, want 409 with explanation:\n%s", rec.Code, rec.Body.String())
			}
			if rec := c.do("GET", p["Cleo"], "en", nil); rec.Code != http.StatusOK {
				t.Errorf("Cleo's link: status %d", rec.Code)
			}
			if strings.Contains(c.do("GET", session, "en", nil).Body.String(), "Dora") {
				t.Error("rejected Save added Dora")
			}
		})
	}
}

func TestSaveExplainsJoinedNameInGerman(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	if body := saveNames(c, session, "de", "Anna", "Ben", "Dora").Body.String(); !strings.Contains(body, "beigetreten") {
		t.Errorf("not German: %s", body)
	}
}

func TestRemovedNamesAreStrippedFromExclusionsAndReAddedStartsClean(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join, _ := createAnnaSession(t, c, "en")
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	ben := c.joinAs(join, "en", joinForm("Ben", "ben@x.de", "w", "Dora"))
	before := len(c.mailer.sent)
	checked := `name="exclude" value="Dora" checked`

	if !strings.Contains(c.do("GET", ben, "en", nil).Body.String(), checked) {
		t.Fatal("Exclusion not set up")
	}
	if rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Emil"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	if page := c.do("GET", ben, "en", nil).Body.String(); strings.Contains(page, `value="Dora"`) {
		t.Errorf("removed name still offered:\n%s", page)
	}
	if n := len(c.mailer.sent) - before; n != 2 { // only the Add email to Anna and Ben
		t.Errorf("%d emails sent", n)
	}
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Emil", "Dora")
	if page := c.do("GET", ben, "en", nil).Body.String(); !strings.Contains(page, `name="exclude" value="Dora"`) || strings.Contains(page, checked) {
		t.Errorf("re-added Dora not clean:\n%s", page)
	}
}

func TestRemovingTheLastBlockerStartsTheDraw(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, p := openSession(t, c)
	saveNames(c, session, "en", "Anna", "Ben", "Cleo", "Dora")
	c.do("POST", p["Anna"], "en", c.personalForm(p["Anna"], "a")) // only Dora blocks
	before := len(c.mailer.sent)
	if rec := saveNames(c, session, "en", "Anna", "Ben", "Cleo"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); !strings.Contains(page, "The names have been drawn.") {
		t.Errorf("not drawn:\n%s", page)
	}
	if n := len(c.mailer.sent) - before; n != 3 {
		t.Errorf("want 3 Draw emails, got %d", n)
	}
}

func TestReleaseThenRemoveWorksForAJoinedName(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, _ := openSession(t, c)
	release(c, session, "Ben")
	before := len(c.mailer.sent)
	if rec := saveNames(c, session, "en", "Anna", "Cleo", "Dora"); rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rec.Code)
	}
	if page := c.do("GET", session, "en", nil).Body.String(); strings.Contains(page, "Ben") {
		t.Errorf("Ben not removed:\n%s", page)
	}
	if len(c.mailer.sent) == before+3 { // Release itself never Draws
		t.Error("unexpected Draw")
	}
}

func TestJoinShowsNotPartOfTheSessionForUnknownNames(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, join := openJoinSession(t, c)
	saveNames(c, session, "en", "Anna", "Cleo", "Dora") // Ben removed
	for _, lang := range []string{"en", "de"} {
		want := map[string]string{"en": "Ben is not part of the session", "de": "Ben gehört nicht zur Runde"}[lang]
		if body := c.do("GET", join+"?name=Ben", lang, nil).Body.String(); !strings.Contains(body, want) || !strings.Contains(body, "Cleo") {
			t.Errorf("%s: join page lacks %q:\n%s", lang, want, body)
		}
		rec := c.do("POST", join, lang, joinForm("Ben", "ben@x.de", "w"))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), want) {
			t.Errorf("%s: join form: status %d, lacks %q:\n%s", lang, rec.Code, want, rec.Body.String())
		}
	}
}
