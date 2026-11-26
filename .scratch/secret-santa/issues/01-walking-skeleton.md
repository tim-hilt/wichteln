# 01: Walking skeleton: welcome page, DE/EN, Docker, CI

**What to build:** A visitor opening the app sees a welcome page offering the three actions: create a new Session, open an existing Session, and open a Personal link. The page is in German or English depending on the browser language, with no language selector. The app is runnable as a tiny `FROM scratch` Docker image, and CI runs the tests and builds the image.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

Spec: `.scratch/secret-santa/spec.md` · Vocabulary: `GLOSSARY.md` · Persistence: ADR-0001

- [ ] Stdlib-only Go module; one application constructor takes the data directory, Mailer, base URL and random source, and returns the root `http.Handler` (Seam A)
- [ ] A Mailer interface exists, with a stdout adapter that logs emails
- [ ] `GET /` renders the welcome page with the three actions (forms may be non-functional for now)
- [ ] All visible text comes from a single DE/EN message catalog that templates use; DE is chosen when German is the top-ranked supported language in `Accept-Language`, otherwise EN
- [ ] Templates, CSS and vendored HTMX are embedded in the binary
- [ ] Config via env: listen address, data directory, base URL
- [ ] Multi-stage Dockerfile whose final stage is `FROM scratch`; the image starts and serves the welcome page
- [ ] A GitHub Actions workflow runs `go test ./...` and builds the Docker image
- [ ] Seam A tests: the welcome page renders in DE for a German `Accept-Language` and in EN otherwise
