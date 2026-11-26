package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestOpenFromWelcomePage(t *testing.T) {
	c := newClient(t, t.TempDir())
	session, personal := openSession(t, c)
	sessionHash := strings.TrimPrefix(session, "/session/")
	anna := personal["Anna"]
	const unknown = "aaaaaaaaaaaaaaaaaaaaaaaaaa"

	tests := []struct {
		name, path, input, wantLocation string
	}{
		{"session bare hash", "/open-session", sessionHash, session},
		{"session full url", "/open-session", " http://example.test" + session + "/ ", session},
		{"link bare hash", "/open-link", anna[1:], anna},
		{"link full url", "/open-link", "http://example.test" + anna, anna},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := c.do("POST", tt.path, "en", url.Values{"hash": {tt.input}})
			if rec.Code != 303 || rec.Header().Get("Location") != tt.wantLocation {
				t.Errorf("status %d, location %q, want %q", rec.Code, rec.Header().Get("Location"), tt.wantLocation)
			}
		})
	}

	for _, path := range []string{"/open-session", "/open-link"} {
		for _, input := range []string{unknown, "http://example.test/session/" + unknown, "garbage", anna[1:] + "x"} {
			rec := c.do("POST", path, "de", url.Values{"hash": {input}})
			body := rec.Body.String()
			if rec.Code != 404 || !strings.Contains(body, "Dieser Link ist ungültig") || !strings.Contains(body, `value="`+input+`"`) {
				t.Errorf("%s %q: status %d, body:\n%s", path, input, rec.Code, body)
			}
		}
	}
	// A Session hash is not a Personal link and vice versa.
	if rec := c.do("POST", "/open-link", "en", url.Values{"hash": {sessionHash}}); rec.Code != 404 {
		t.Errorf("session hash opened as link: %d", rec.Code)
	}
	if rec := c.do("POST", "/open-session", "en", url.Values{"hash": {anna[1:]}}); rec.Code != 404 {
		t.Errorf("link hash opened as session: %d", rec.Code)
	}
}

func TestJoinFromWelcomePage(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join, _ := createAnnaSession(t, c, "en")
	hash := strings.TrimPrefix(join, "/join/")

	for _, input := range []string{hash, "http://example.test" + join + "/", "http://example.test" + join + "?name=Ben#x"} {
		rec := c.do("POST", "/open-join", "en", url.Values{"hash": {input}})
		if rec.Code != 303 || rec.Header().Get("Location") != join {
			t.Errorf("%q: status %d, location %q, want %q", input, rec.Code, rec.Header().Get("Location"), join)
		}
	}

	for _, input := range []string{"aaaaaaaaaaaaaaaaaaaaaaaaaa", "garbage"} {
		rec := c.do("POST", "/open-join", "de", url.Values{"hash": {input}})
		body := rec.Body.String()
		if rec.Code != 404 || !strings.Contains(body, "Dieser Link ist ungültig") || !strings.Contains(body, `value="`+input+`"`) {
			t.Errorf("%q: status %d, body:\n%s", input, rec.Code, body)
		}
	}

	// A Join hash is not a Personal link.
	if rec := c.do("POST", "/open-link", "en", url.Values{"hash": {hash}}); rec.Code != 404 {
		t.Errorf("join hash opened as link: %d", rec.Code)
	}
}

func TestJoinFromWelcomePageAfterDrawRedirectsToClosedPage(t *testing.T) {
	c := newClient(t, t.TempDir())
	_, join, anna := createAnnaSession(t, c, "en")
	c.do("POST", join, "en", joinForm("Ben", "ben@x.de", "b"))
	c.do("POST", join, "en", joinForm("Cleo", "cleo@x.de", "c"))
	c.do("POST", anna, "en", c.personalForm(anna, "a"))

	rec := c.do("POST", "/open-join", "en", url.Values{"hash": {strings.TrimPrefix(join, "/join/")}})
	if rec.Code != 303 || rec.Header().Get("Location") != join {
		t.Fatalf("status %d, location %q", rec.Code, rec.Header().Get("Location"))
	}
	if body := c.do("GET", join, "en", nil).Body.String(); !strings.Contains(body, "You can no longer join") {
		t.Errorf("want closed page:\n%s", body)
	}
}
