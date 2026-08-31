# End-to-end tests

Browser tests that drive the **real** autobrr binary through its web UI with
[playwright-go](https://github.com/mxschmitt/playwright-go), including a full
pass down the pipeline autobrr exists for: an IRC announce is matched by a
filter and handed to an action.

Everything here is behind the `e2e` build tag, so it is compiled out of normal
builds and never reaches the shipped binary. Like `test/irc`, it is part of the
main module and needs no Docker, no external services and no CI choreography:
the tests compile and start the processes they need themselves.

## Running

The browser driver has to be installed once. It is ~150 MB and lands in
`~/.cache/ms-playwright`:

```sh
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 install --with-deps chromium
```

The tests serve the UI from the binary, so the frontend has to be built first:

```sh
cd web && pnpm install --frozen-lockfile && pnpm run build
```

Then:

```sh
# the whole suite:
go test -tags=e2e ./test/e2e/...

# one test:
go test -tags=e2e -run TestAnnounceMatchesFilterAndRunsAction ./test/e2e/...

# watch it happen in a real browser window:
HEADED=1 go test -tags=e2e -run TestAddIndexer ./test/e2e/...
```

Without the tag these packages contain no buildable files, so `go build ./...`,
`go vet ./...` and a plain `go test ./...` skip them entirely. As with
`test/irc`, tooling that discovers packages first has to pass the tag to
`go list` too.

## When a test fails

Every browser context is traced. On failure the trace and a full-page
screenshot are written to `test/e2e/artifacts/`, and both paths are printed in
the test output along with the autobrr and mock indexer logs. The trace is the
useful one:

```sh
go run github.com/mxschmitt/playwright-go/cmd/playwright@v0.6100.0 show-trace \
  test/e2e/artifacts/TestAddIndexer.trace.zip
```

It replays the run with the DOM at every step, so "a locator timed out" becomes
a picture of the page at the moment it gave up. CI uploads the directory as an
artifact on failure.

## Layout

- `harness/` - everything that is not an assertion.
  - `harness.go` compiles autobrr and the mock indexer and launches the browser,
    once for the package.
  - `app.go` starts one autobrr per test on a free port with its own config
    directory and database, and onboards and authenticates over the API.
  - `browser.go` creates a per-test browser context: locale pinned, session
    cookie injected, tracing on, artifacts saved if the test fails.
  - `ui.go` wraps the page in the interactions the UI needs. The web UI has
    three different dropdown implementations, so there are three ways to pick
    an option.
  - `mockindexer.go`/`torrent.go` run the mock indexer with a generated,
    genuinely valid torrent for it to serve.
  - `irc.go` waits on autobrr's own view of its IRC connection.
- `*_test.go` - the scenarios.

## Conventions worth keeping

**One instance per test.** `harness.Start(t)` gives each test its own process,
config and database. Tests do not share state and do not have to run in order,
so a failure points at one thing.

**No sleeps.** Playwright locators already wait for an element to be attached,
visible, stable and able to receive events. A fixed sleep only makes a passing
test slower and a failing one flakier. Where the wait is on autobrr rather than
the DOM - has it joined the IRC channel yet - ask it over the API instead
(`WaitForChannelMonitoring`).

**Fatal, not accumulating.** The harness fails the test on the first broken
step. There is no sensible assertion to make after a form failed to open, and
continuing turns one real failure into a page of noise.

**Ids over text.** The shared input components set `id` and `name` from the
Formik field name, which is stable in a way that visible copy is not. The UI is
translated into eight languages and picks one from `navigator.language`, so the
browser context pins `en-US`; the remaining text selectors are button captions
and toasts, which fail loudly and obviously when the copy changes.

## Adding a scenario

```go
func TestSomething(t *testing.T) {
	app := harness.Start(t)
	page := app.NewAuthedPage()

	page.Open("/settings/indexers")
	page.AddNew()
	page.Select("Generic RSS")
	page.Fill("feed.url", "https://example.com/rss")
	page.Submit()

	page.ExpectRow("Generic RSS")
}
```

If a scenario needs an announce, add `harness.StartMockIndexer(t)`. Its ports
are fixed by `test/definitions/mock.yaml`, so a locally running mock indexer
has to be stopped first, and these tests do not use `t.Parallel()`.
