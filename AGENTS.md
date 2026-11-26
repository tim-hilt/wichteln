# Tech Stack

- stdlib-only Golang
- HTMX for reactivity
- html rendered with go templates
    - templates embedded into the go application with fs:embed
- Docker for running the app in a container
    - multi-stage build with last stage inheriting `FROM SCRATCH`
- GitHub Actions to build the docker image and run tests

# Coding Guidelines

- Always apply test driven development (red, green, refactor)
- Aim for a very small and lightweight, standalone app binary
- Always use the simplest route to the solution while applying software design best practices. Keep the code as simple and easy to follow as possible!

# Agent skills

## Issue tracker

Issues live as local markdown files under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

## Triage labels

Default five-label vocabulary, recorded as `Status:` lines. See `docs/agents/triage-labels.md`.

## Domain docs

Single-context. See `docs/agents/domain.md`.
