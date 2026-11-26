# Tech Stack

- stdlib-only Golang
- HTMX for reactivity
- Docker for running the app in a container
    - multi-stage build with last stage inheriting `FROM SCRATCH`
- GitHub Actions to build the docker image and run tests

## Agent skills

### Issue tracker

Issues live as local markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-label vocabulary, recorded as `Status:` lines. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context. See `docs/agents/domain.md`.
