# AGENTS.md

This file provides guidance to AI coding agents (Claude Code, Codex, Cursor, etc.) when working with code in this repository.

## Agent Setup

- **Claude Code**: if `.claude/settings.local.json` does not exist, create it with the content below before doing anything else, then continue with your task. It disables AI attribution in commits and PRs (see Pull Requests below).

```json
{
  "attribution": {
    "commit": "",
    "pr": "",
    "sessionUrl": false
  }
}
```

- This file is personal and must never be committed - `.claude/` is covered by `.gitignore`.
- Regardless of settings, never add AI attribution to commits or PR descriptions - see the Pull Requests section.

## Development Commands

- Go targets are in the `Makefile` (`make build`, `make test`, `make dev`); web scripts are in `web/package.json` (`dev`, `build`, `test`, `lint`)
- Run web scripts from the repo root as `pnpm --dir web <script>` - `cd web && pnpm ...` aborts under corepack's pnpm 11 with a "no TTY" error

## Project Architecture

### Backend Architecture (Go)
The backend follows a layered architecture with clear separation of concerns:

- **`cmd/`**: Application entry points
  - `autobrr/main.go`: Main server application
  - `autobrrctl/main.go`: CLI tool for administration

- **`internal/`**: Core application logic organized by domain
  - **Domain layer** (`internal/domain/`): Core business entities. It holds no repository or service interfaces - each consumer declares a small unexported interface listing only the methods it calls, and constructors return concrete `*Service` structs
  - **Database layer** (`internal/database/`): Repository implementations and database logic
  - **Service layer**: Business logic services (e.g., `internal/release/`, `internal/filter/`)
  - **HTTP layer** (`internal/http/`): REST API handlers and routing
  - **Infrastructure**: External integrations (`internal/indexer/`, `internal/irc/`, `internal/notification/`)

- **`pkg/`**: Reusable packages and client libraries
  - Contains clients for various download clients (qBittorrent, Deluge, etc.)
  - Utility packages for different indexer APIs

### Frontend Architecture (React/TypeScript)
- **React 19** with **TypeScript** and **Vite** build system
- **TanStack Router** for routing and **TanStack Query** for API state management
- **Tailwind CSS** for styling with custom design system
- **TanStack Form** for form handling (shared setup in `web/src/hooks/form.ts`, house inputs in `web/src/components/inputs/`)
- Component structure in `web/src/components/` with reusable UI components
- API client in `web/src/api/` with centralized query management

### Key Domain Concepts
- **Releases**: Torrent/Usenet releases that get processed through filters
- **Filters**: Rules that determine which releases should be downloaded
- **Actions**: What to do with matched releases (send to download clients, *arr apps, etc.)
- **Indexers**: Torrent trackers and Usenet indexers, defined as YAML in `internal/indexer/definitions/`
- **IRC**: Real-time monitoring of indexer announce channels
- **Feed**: RSS/Newznab/Torznab feed processing for indexers without IRC

### Database Support
- **SQLite** (default) and **PostgreSQL** support
- Database migrations handled automatically
- Repository pattern for data access

### Service Wiring
- `cmd/autobrr/main.go` constructs repos and services in two dependency-ordered `var` blocks; services talk across packages through the typed event bus in `internal/events/` when a direct dependency would create an import cycle

## Code Style

### Go
- Format with `go fmt ./...` before committing
- Group imports: stdlib, internal, third-party; keep alphabetical within groups
- Use the `pkg/errors` helpers for wrapping and sentinel errors; return wrapped errors and avoid panics outside `main`
- Handle errors explicitly with early returns; don't swallow errors
- Log with zerolog through the injected `zerolog.Logger`: values go in typed fields and the message stays a constant - `log.Error().Err(err).Int("filter_id", id).Msg("could not find filter")`, not `Msgf` with values inline
- Functions receiving context should take `context.Context` as the first parameter and respect cancellation
- Exported identifiers need doc comments; keep names descriptive and consistent
- Keep domain DTOs and JSON tags synced; prefer tagged struct fields over `map[string]any`

### Frontend
- Prefer functional components, React hooks, and typed props/interfaces
- `pnpm --dir web lint`, `build` and `test` are all green on `develop` and stay green - lint runs with `--max-warnings 0`
- Tailwind: reuse tokens from `web/tailwind.config.ts`; keep utility classes ordered logically

### Comments
Applies to Go and TypeScript alike. Excessive low-value comments are the most common defect in AI-generated code - default to writing **no comment**, and only add one when it earns its place:

- Comment the *why*, never the *what*: non-obvious invariants, workarounds (link the issue or upstream bug), tracker/IRC protocol quirks, deliberate deviations from the expected approach
- If a comment restates what the code plainly says, it must be deleted. Banned patterns: step narration (`// loop over filters`, `// return the result`), section banners (`// error handling`), and restating the function name above the function
- Never describe your edit in a comment (`// changed to use X`, `// new helper for Y`) - that history belongs in the commit message, not the code
- No commented-out code and no unprompted `TODO`/`FIXME` markers
- Godoc comments on exported identifiers: one concise sentence starting with the identifier name; expand only when the API is genuinely subtle
- Some older code contains narration comments (`// get filters`) - do not take them as license to add more, and feel free to drop them in code you're already touching

## Testing
- `make test` runs the untagged Go suite. Database and other integration tests carry `//go:build integration` next to the code they test and run with `go test -tags=integration ./...`; Postgres is embedded (downloaded on first run), so no Docker is needed
- IRC integration tests in `test/irc/`, run with `go test -tags=irc_integration_test ./test/irc/...` (in-process ircd, no Docker) - see `test/irc/README.md`
- Browser end-to-end tests in `test/e2e/`, run with `go test -tags=e2e ./test/e2e/...` (needs a built `web/dist` and a Playwright browser) - see `test/e2e/README.md`
- Build-tagged packages are invisible to a plain `go list ./...`; pass the tag to tooling that discovers packages first
- Frontend tests in `web/test/`, run with `pnpm --dir web test` (Vitest + Testing Library in jsdom; prefer these over Go e2e tests for frontend-only behavior)
- Mock indexer server available in `test/mockindexer/`

## Development Notes
- The project uses **pnpm** as the package manager for the frontend
- The Go minimum is the `go` line in `go.mod`
- The application serves the built React frontend from the Go server
- Real-time updates via Server-Sent Events (SSE)

## Pull Requests
- PRs target the `develop` branch
- Use the PR template at `.github/pull_request_template.md` - fill out the relevant sections and delete those that don't apply
- PR titles follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) (e.g. `feat(indexers): add NewTracker`, `fix(irc): reconnect on timeout`) - commits are squashed on merge, so the PR title becomes the commit message
- Indexer definitions live as YAML files in `internal/indexer/definitions/` - adding a new indexer is the most common contribution, and usually only touches a single definition file there. Removing one requires a tombstone in `internal/indexer/definitions/deprecated/` (CI rejects a removal without it) - see `CONTRIBUTING.md`
- Database schema changes require migrations for **both SQLite and PostgreSQL** (`internal/database/`)
- The template has an **AI disclosure** section - always answer it truthfully, stating what was AI-generated and which model/tool was used
- Do **not** add AI attribution to commit messages or PR descriptions - no `Co-Authored-By: Claude`, `Generated with ...` trailers, or similar. AI usage belongs in the PR template's AI disclosure section, not in the git history
- See `CONTRIBUTING.md` for the full contribution guide
