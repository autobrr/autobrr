# Simulation

Runs the real autobrr binary against a set of mock indexers and download
clients, delivers 50-100 releases through IRC and feeds, and checks that every
release ended up where the filters say it should, with the right Discord
notification.

Everything runs in-process apart from autobrr itself, so there is no Docker and
nothing to start by hand. The package is behind the `simulation` build tag.

## Running

```sh
make test/simulation

# or with flags
make test/simulation SIMFLAGS="-releases=100 -seed=42"
go test -tags=simulation -count=1 -v ./test/simulation/ -releases=60 -interval=0
```

It can also be started from the Actions tab as the **Simulation** workflow
(`workflow_dispatch`). The report is written to the job summary, and the report
files and autobrr log are uploaded as an artifact.

| flag           | default    | what it does                                                  |
|----------------|------------|---------------------------------------------------------------|
| `-releases`    | 80         | number of releases to deliver                                 |
| `-seed`        | 0          | generator seed, 0 picks one; the report prints it for replays |
| `-interval`    | 25ms       | pause between releases, 0 sends them as a burst               |
| `-feed-batch`  | 5          | feed items published per forced feed run                      |
| `-wait`        | 2m         | how long to wait for the expected outcomes                    |
| `-settle`      | 3s         | extra wait afterwards, to catch duplicate pushes              |
| `-out`         | temp dir   | where `autobrr.log`, `report.md` and `report.json` go; the temp dir is kept when the run fails |
| `-autobrr-bin` | build      | test a prebuilt binary instead of building `./cmd/autobrr`    |
| `-keep`        | false      | leave autobrr and the mocks up after the run, to poke at them |
| `-log-level`   | DEBUG      | autobrr's log level                                           |
| `-pgo-profile` | off        | write autobrr's CPU profile for the run here, see below       |

With `-keep` the run blocks after the report until it is interrupted, and logs
the url and credentials. The UI is only there if `web/dist` was built before
autobrr was compiled (`pnpm --dir web build`). Plain `go test` needs
`-timeout 0` with `-keep`, the make target already passes it.

## CPU profile for PGO

`-pgo-profile=cpu.pprof` turns on autobrr's pprof server and records CPU from
the first delivered release until verification ends, so the profile covers the
announce, filter and action path rather than startup. The endpoint needs its
duration up front, so it is recorded in 2s chunks and merged with
`go tool pprof -proto`. A useful profile needs a big run at the log level real
installs use:

```sh
make test/simulation SIMFLAGS="-releases=2000 -interval=0 -feed-batch=25 -log-level=INFO -pgo-profile=$PWD/cpu.pprof"
go build -pgo=cpu.pprof ./cmd/autobrr
```

autobrr's total CPU time is logged when it shuts down, which makes it easy to
compare builds on the same seed with `-autobrr-bin`.

## What runs

| mock               | kind    | details                                                                 |
|--------------------|---------|-------------------------------------------------------------------------|
| SimAlpha, SimBravo | IRC     | one in-process ircd each (`test/irc/ircd`), custom definitions generated at runtime |
| SimCharlie RSS     | RSS     | Generic RSS indexer + feed                                              |
| SimDelta Torznab   | Torznab | Generic Torznab indexer + feed, freeleech via `downloadvolumefactor`    |
| qBittorrent        | client  | login, `torrents/add`; actions skip reannounce, `torrents/trackers` reports a working tracker if one is enabled |
| Sonarr, Radarr     | client  | `release/push`; they fetch the download url like a real arr does, and reject group `RJCT` |
| Discord            | notify  | `test/mockdiscord`, validates payloads against Discord's limits         |

Every indexer has a tracker that serves real torrents behind a passkey, so a
broken download url or a lost passkey fails the run.

The filters, highest priority first:

| filter         | indexers         | catches                          | action                      |
|----------------|------------------|----------------------------------|-----------------------------|
| TV HD          | all              | the sim shows in 1080p           | Sonarr                      |
| Movies UHD     | all              | 2160p movies                     | Radarr                      |
| TV Fallback    | all              | the sim shows in any resolution  | qBittorrent `tv-fallback`   |
| Linux ISOs     | IRC only         | `*Linux*ISO*`                    | qBittorrent `isos`          |
| Freeleech FLAC | alpha, Torznab   | freeleech `*FLAC*`               | qBittorrent `music-fl`      |

TV HD releases from group `RJCT` are rejected by Sonarr and have to fall through
to TV Fallback, which exercises autobrr moving on to the next filter after an
arr rejection. ISOs from the feeds and FLAC from RSS must match nothing.

## How it decides pass or fail

Each generated release carries its ground truth (TV HD, 2160p movie, ISO, ...).
The harness walks the filters in priority order with simple predicates over that
ground truth instead of reimplementing autobrr's matching, which gives the exact
pushes and notifications to expect. After delivery it waits until the mocks have
seen all of them, waits `-settle` more for stragglers, and diffs:

- **missing** - expected but never observed
- **unexpected** - observed but not expected, duplicates included

Either one, or not settling within `-wait`, fails the test. The report also has
per-source latency from delivery to first push, and warnings for passkey
denials, re-downloads and requests to mock endpoints that are not implemented,
which usually means autobrr started calling something new.

## Extending it

- New release shape: add a `Kind` and its name format in `releases.go`.
- New filter: add a `FilterSpec` in `scenario.go`. `Configure` edits the filter
  json the same way the UI form would, `Matches` is its ground-truth predicate.
- New client: add a mock next to `clients.go`, a downloader in
  `Scenario.Configure`, and its outcomes in `expectedOutcomes`/`observed`.

Request bodies are plain maps on purpose. The domain types redact their secrets
in `MarshalJSON`, so encoding one sends `<redacted>` as the passkey.
