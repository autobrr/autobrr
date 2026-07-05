# IRC integration tests

End-to-end tests for `internal/irc` that run the **real** `irc.Handler` against an
**in-process** IRC server over a loopback socket - no Docker, no external binary.

Everything here - the test server and the tests - is behind the
`irc_integration_test` build tag, so it is compiled out of normal builds and
never reaches the shipped autobrr binary. It is part of the main module (no extra
dependencies over what autobrr already uses), so it runs with a single tagged
`go test`.

## Running

```sh
# just the irc integration tests:
go test -tags=irc_integration_test ./test/irc/...
# with the race detector:
go test -tags=irc_integration_test -race ./test/irc/...
# as part of the whole suite:
go test -tags=irc_integration_test ./...
```

Without the tag these packages contain no buildable files, so `go build ./...`,
`go vet ./...` and a plain `go test ./...` skip them entirely.

Note: `go list`/`go test` only *see* these packages when the tag is set — a
plain `go list ./...` will not enumerate them. Tooling that discovers packages
first (e.g. a per-package profiling script) must pass the tag to `go list` too:
`go list -tags=irc_integration_test ./...`.

## Layout

- `ircd/` - the minimal test IRC server. It implements just enough of the protocol
  for the `ergochat/irc-go` client to register and operate: CAP/SASL PLAIN
  negotiation, NickServ IDENTIFY, channel modes `+k` (key), `+i` (invite-only) and
  `+r` (registered-only), auditorium-style hidden announcers, and **virtual bots**
  that stand in for tracker announcers and invite gatekeepers (scripted via
  callbacks - no second client to manage).
- `harness/` - wires a real `irc.Handler` up to the server: it supplies the
  collaborators `NewHandler` needs (SSE sink, release sink, no-op notifier), drives
  the connection lifecycle, and exposes helpers to wait on channel state
  (`WaitForMonitoring`, `WaitForState`) and captured releases. It also builds
  minimal indexer definitions so a scripted announce parses into a release.
- `*_test.go` - the scenarios, reproducing the per-tracker setups from
  `irc_test.md` (auth mechanism, channel modes, invite command).

## What's covered

- **Auth**: NONE (direct join), SASL PLAIN, NickServ IDENTIFY - the latter two gate
  a `+r` channel so reaching `Monitoring` proves authentication worked.
- **Channel modes**: `+k` bad/missing key → `475` surfaced as a channel error, `+k`
  with the right key → joins, `+i` without an invite → `473`.
- **Invite flow**: gatekeeper accepts (INVITE → join), gatekeeper rejects (parks in
  `InviteFailed` with the reason, no retry), gatekeeper absent (`401` → keeps
  retrying via backoff), and late accept after a rejection (recovers).
- **Kick**: a `KICK` parks the channel in `Kicked` and it is **not** auto-rejoined
  (asserted via the server's join counter).

## Adding a scenario

```go
srv := ircd.New(t)
srv.AddChannel("#announce", ircd.Key("s3cr3t"), ircd.Announcer("Bot"))

def := harness.MinimalDefinition("mytracker", "#announce", "Bot")
net := harness.Network(srv, "mynick", harness.SASL("acct", "pass"),
    harness.ChannelWithPassword("#announce", "s3cr3t"))

inst := harness.Start(t, net, harness.Defs(def))
inst.WaitForMonitoring("#announce", 10*time.Second)

srv.Announce("#announce", "Bot", "New torrent: Some.Release in Movies")
rls, ok := inst.Releases.Wait(5 * time.Second)
```

Pass `harness.Options{Verbose: true}` to `Start` to stream the handler's trace log
(and the raw IRC lines) to the test output when debugging.
