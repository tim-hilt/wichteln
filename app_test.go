package main

import (
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path, acceptLanguage string) string {
	t.Helper()
	app := NewApp(t.TempDir(), StdoutMailer{Out: io.Discard}, "http://localhost", rand.Reader)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d", path, rec.Code)
	}
	return rec.Body.String()
}

func TestWelcomePageLanguage(t *testing.T) {
	const de = "Neue Sitzung erstellen"
	const en = "Create a new Session"
	tests := []struct {
		name, header, want string
	}{
		{"german", "de-DE,de;q=0.9,en;q=0.8", de},
		{"german over unsupported", "fr,de;q=0.5", de},
		{"english preferred over german", "en-US,en;q=0.9,de;q=0.8", en},
		{"no header", "", en},
		{"unsupported only", "fr-FR", en},
		{"german rejected", "de;q=0, en", en},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := get(t, "/", tt.header)
			if !strings.Contains(body, tt.want) {
				t.Errorf("want %q in page:\n%s", tt.want, body)
			}
			other := de
			if tt.want == de {
				other = en
			}
			if strings.Contains(body, other) {
				t.Errorf("unexpected %q in page", other)
			}
		})
	}
}

func TestWelcomePageOffersThreeActions(t *testing.T) {
	body := get(t, "/", "en")
	for _, want := range []string{"Create a new Session", "Open an existing Session", "Open a Personal link"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestStaticAssetsServed(t *testing.T) {
	for _, p := range []string{"/static/htmx.min.js", "/static/style.css"} {
		if get(t, p, "") == "" {
			t.Errorf("%s empty", p)
		}
	}
}

func TestStdoutMailerLogsEmail(t *testing.T) {
	var sb strings.Builder
	err := StdoutMailer{Out: &sb}.Send(Mail{To: "a@b.de", Subject: "Hi", Text: "Body", HTML: "<p>Body</p>"})
	if err != nil || !strings.Contains(sb.String(), "a@b.de") || !strings.Contains(sb.String(), "Hi") {
		t.Errorf("err=%v out=%q", err, sb.String())
	}
}

func TestCatalogsDefineSameKeys(t *testing.T) {
	for key := range catalog["en"] {
		if _, ok := catalog["de"][key]; !ok {
			t.Errorf("de missing %q", key)
		}
	}
	for key := range catalog["de"] {
		if _, ok := catalog["en"][key]; !ok {
			t.Errorf("en missing %q", key)
		}
	}
}
