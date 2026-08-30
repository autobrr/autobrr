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

### Backend (Go)
- **Build**: `make build` - Builds both web frontend and Go backend
- **Build backend only**: `make build/app` - Builds the main autobrr binary
- **Build CLI tool**: `make build/ctl` - Builds the autobrrctl binary
- **Test**: `make test` - Runs Go tests (excludes integration tests)
- **Run a single test**: `go test ./path/to/pkg -run TestName`
- **Install dependencies**: `make deps` - Installs both Go and web dependencies
- **Development mode**: `make dev` - Starts both frontend dev server and backend in tmux session

### Frontend (React/TypeScript)
- **Development server**: `pnpm --dir web dev` - Starts Vite dev server
- **Build**: `pnpm --dir web build` - TypeScript compilation and Vite build
- **Test**: `pnpm --dir web test` - Vitest unit/component tests (`test:watch` for watch mode)
- **Lint**: `pnpm --dir web lint` - ESLint check
- **Lint with watch**: `pnpm --dir web lint:watch` - ESLint in watch mode

### Docker
- **Build image**: `make build/docker` - Builds development Docker image

## Project Architecture

### Backend Architecture (Go)
The backend follows a layered architecture with clear separation of concerns:

- **`cmd/`**: Application entry points
  - `autobrr/main.go`: Main server application
  - `autobrrctl/main.go`: CLI tool for administration

- **`internal/`**: Core application logic organized by domain
  - **Domain layer** (`internal/domain/`): Core business entities and interfaces
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
- **Formik** and **React Hook Form** for form handling
- Component structure in `web/src/components/` with reusable UI components
- API client in `web/src/api/` with centralized query management

### Key Domain Concepts
- **Releases**: Torrent/Usenet releases that get processed through filters
- **Filters**: Rules that determine which releases should be downloaded
- **Actions**: What to do with matched releases (send to download clients, *arr apps, etc.)
- **Indexers**: Torrent trackers and Usenet indexers (75+ supported via IRC announces)
- **IRC**: Real-time monitoring of indexer announce channels
- **Feed**: RSS/Newznab/Torznab feed processing for indexers without IRC

### Database Support
- **SQLite** (default) and **PostgreSQL** support
- Database migrations handled automatically
- Repository pattern for data access

### Service Dependencies
The main application (`cmd/autobrr/main.go`) orchestrates these key services:
- **IRC Service**: Monitors IRC channels for torrent announces
- **Feed Service**: Processes RSS/Newznab feeds
- **Filter Service**: Applies user-defined rules to releases
- **Release Service**: Manages release lifecycle and processing
- **Action Service**: Executes actions on matched releases
- **Indexer Service**: Manages indexer definitions and API interactions
- **Download Client Service**: Interfaces with various download clients

### Configuration
- Configuration via `config.toml` file or environment variables
- Dynamic configuration reloading supported
- Extensive environment variable support (see README.md for full list)

## Code Style

### Go
- Format with `go fmt ./...` before committing
- Group imports: stdlib, internal, third-party; keep alphabetical within groups
- Use the `pkg/errors` helpers for wrapping and sentinel errors; return wrapped errors and avoid panics outside `main`
- Handle errors explicitly with early returns; don't swallow errors
- Use `logger.Logger`/zerolog for structured logging; avoid bare `fmt.Println`
- Functions receiving context should take `context.Context` as the first parameter and respect cancellation
- Exported identifiers need doc comments; keep names descriptive and consistent
- Keep domain DTOs and JSON tags synced; prefer tagged struct fields over `map[string]any`

### Frontend
- Prefer functional components, React hooks, and typed props/interfaces
- Run `pnpm --dir web lint` and `pnpm --dir web build` for diagnostics before shipping
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
- Go tests exclude integration tests by default: `go test $(go list ./... | grep -v test/integration)`
- Integration tests available in `test/integration/`, run with `go test ./... -tags=integration` (Postgres tests require Docker)
- IRC integration tests in `test/irc/`, run with `go test -tags=irc_integration_test ./test/irc/...` (in-process ircd, no Docker) - see `test/irc/README.md`
- Browser end-to-end tests in `test/e2e/`, run with `go test -tags=e2e ./test/e2e/...` (needs a built `web/dist` and a Playwright browser) - see `test/e2e/README.md`
- Build-tagged packages are invisible to a plain `go list ./...`; pass the tag to tooling that discovers packages first
- Frontend tests in `web/test/`, run with `pnpm --dir web test` (Vitest + Testing Library in jsdom; prefer these over Go e2e tests for frontend-only behavior)
- Mock indexer server available in `test/mockindexer/`

## Development Notes
- The project uses **pnpm** as the package manager for the frontend
- Go version 1.26+ required (see `go.mod` for the current minimum)
- The application serves the built React frontend from the Go server
- Real-time updates via Server-Sent Events (SSE)

## Pull Requests
- PRs target the `develop` branch
- Use the PR template at `.github/pull_request_template.md` - fill out the relevant sections and delete those that don't apply
- PR titles follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) (e.g. `feat(indexers): add NewTracker`, `fix(irc): reconnect on timeout`) - commits are squashed on merge, so the PR title becomes the commit message
- Indexer definitions live as YAML files in `internal/indexer/definitions/` - adding a new indexer is the most common contribution, and usually only touches a single definition file there
- Database schema changes require migrations for **both SQLite and PostgreSQL** (`internal/database/`)
- The template has an **AI disclosure** section - always answer it truthfully, stating what was AI-generated and which model/tool was used
- Do **not** add AI attribution to commit messages or PR descriptions - no `Co-Authored-By: Claude`, `Generated with ...` trailers, or similar. AI usage belongs in the PR template's AI disclosure section, not in the git history
- See `CONTRIBUTING.md` for the full contribution guide
