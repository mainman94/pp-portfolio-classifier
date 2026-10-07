# AGENTS

Go CLI that reads a Portfolio Performance XML file, fetches Morningstar
classification data, and writes taxonomies back into a new XML plus a CSV dump.
README.md documents every flag.

## Workflow

```bash
go build -o pp-classifier .
go vet ./...
go test ./...
```

There is no CI. Run `go vet` and `go test` before every commit.

## Layout

- `main.go` → `internal/app`: CLI flow.
- `internal/config`: flag parsing (Python-style argument order is supported).
- `internal/morningstar`: HTTP client. Auth uses a bearer token (`-token` or
  `MS_TOKEN`). The legacy page scrape no longer works.
- `internal/ppxml`: read/write the Portfolio Performance XML.
- `internal/taxonomy`: taxonomy definitions. `internal/crypto`: crypto fallback.

## Rules

- **Portfolio XML and CSV files hold personal financial data.** `*.xml` and
  `*.csv` are gitignored. Never commit them, never paste their content into
  issues, commits, or test fixtures. Tests use synthetic data only.
- Never hard-code a Morningstar token. It comes from `-token` or `MS_TOKEN`.
- Tests must not call Morningstar. Stub HTTP with the `roundTripFunc` / `testClient` helpers in
  `internal/morningstar/client_test.go`.
