# Repository Guidelines

## Project Structure & Module Organization
- Entry point lives in `main.go`; cross-cutting logic sits under `server`, `router`, and `exchange`.
- Bidder adapters are in `adapters/` (with adapter-specific tests alongside). Auction modules live in `modules/`. Endpoint logic is under `endpoints/`. Shared utilities are in `util/`, `pbs/`, and `openrtb_ext/`.
- Configuration, samples, and docs: `config/`, `sample/`, `docs/`. Static assets required at startup are in `static/`.

## Build, Test, and Development Commands
- `go run .` — start the server locally (defaults to port 8000).
- `./validate.sh` — format, run Go tests (excludes vendor), and vet (pass `--nofmt`, `--cov`, `--race <n>`, or `--novet` as needed).
- `make test` — vendor deps and run `./validate.sh`. Use `make build` to run tests then build the binary.
- `make build-modules` — regenerate `modules/builder.go` after adding/updating modules.
- `go test github.com/prebid/prebid-server/v3/adapters/<adapter> -bench=.` — adapter-specific quick check.

## Coding Style & Naming Conventions
- Go 1.23+ with `cgo` enabled (default). Use standard Go formatting; `./scripts/format.sh -f true` or `make format` auto-applies `gofmt` rules.
- Follow Effective Go practices: prefer small, descriptive functions and handle errors rather than discarding them.
- Package names are lower_snake without plurals; exported identifiers use PascalCase, locals use mixedCase; constants prefer ALL_CAPS when acting as enums.

## Testing Guidelines
- Tests live beside source as `*_test.go`; use Go’s standard `testing` package.
- Prefer table-driven tests and clear, scenario-based names (e.g., `Test[Component]_[Condition]`).
- Run `./validate.sh --cov` when adding significant logic; add race runs (`--race 50`) for concurrency-sensitive changes.

## Commit & Pull Request Guidelines
- Commit messages are short and imperative (e.g., “add ssp specific floors resolution”, “refactor backoff calculation...”); keep scope focused.
- Before opening a PR: run `./validate.sh`; describe behavior changes, configuration impacts, and include links to related issues or specs.
- For features touching adapters/modules, mention impacted bidder/module names and any new config flags. Include screenshots or sample requests/responses when changing endpoint behavior.

## Security & Configuration Tips
- Always set a default GDPR value in config; avoid shipping with unset regulatory defaults.
- Ensure `static/` ships with deployments; missing assets will block startup.
- When adding native modules, document extra system deps (`libatomic`, compiler) and keep `go.mod`/`go.sum` tidy (`go mod tidy`, `go mod vendor` via `make test`).
