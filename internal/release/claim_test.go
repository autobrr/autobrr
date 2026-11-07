// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package release

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const claimExactProfileID = 1

var (
	claimPGPort int
	claimDBSeq  atomic.Int64
)

func TestMain(m *testing.M) {
	os.Exit(runWithEmbeddedPG(m))
}

// runWithEmbeddedPG starts one Postgres server for the package; every test env gets its own
// database on it.
func runWithEmbeddedPG(m *testing.M) int {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not find a free port:", err)
		return 1
	}
	claimPGPort = l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	runtimePath, err := os.MkdirTemp("", "autobrr-release-pg")
	if err != nil {
		fmt.Fprintln(os.Stderr, "could not create postgres runtime dir:", err)
		return 1
	}
	defer os.RemoveAll(runtimePath)

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Database("autobrr").
		Port(uint32(claimPGPort)).
		Version(embeddedpostgres.V17).
		BinariesPath(filepath.Join(cacheDir, "autobrr", "embedded-postgres", "v17-release")).
		RuntimePath(runtimePath).
		StartTimeout(45 * time.Second).
		Logger(&bytes.Buffer{}))

	if err := pg.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "could not start embedded postgres:", err)
		return 1
	}
	defer func() { _ = pg.Stop() }()

	return m.Run()
}

func openClaimTestDB(t *testing.T, engine string) *database.DB {
	t.Helper()
	t.Setenv("IS_TEST_ENV", "true")

	log := logger.Mock()
	cfg := &domain.Config{LogLevel: "ERROR", DatabaseType: "sqlite", DatabaseDSN: ":memory:"}

	if engine == "postgres" {
		admin, err := database.NewDB(&domain.Config{LogLevel: "ERROR", DatabaseType: "postgres", DatabaseDSN: fmt.Sprintf("postgres://postgres:postgres@localhost:%d/autobrr?sslmode=disable", claimPGPort)}, log)
		require.NoError(t, err)
		require.NoError(t, admin.Open())

		name := fmt.Sprintf("claim_%d", claimDBSeq.Add(1))
		_, err = admin.Handler.ExecContext(t.Context(), "CREATE DATABASE "+name)
		require.NoError(t, err)
		require.NoError(t, admin.Close())

		cfg = &domain.Config{LogLevel: "ERROR", DatabaseType: "postgres", DatabaseDSN: fmt.Sprintf("postgres://postgres:postgres@localhost:%d/%s?sslmode=disable", claimPGPort, name)}
	}

	db, err := database.NewDB(cfg, log)
	require.NoError(t, err)
	require.NoError(t, db.Open())
	require.NoError(t, db.Migrate())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func forEachEngine(t *testing.T, fn func(t *testing.T, engine string)) {
	for _, engine := range []string{"sqlite", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			fn(t, engine)
		})
	}
}

type claimTestBus struct{}

func (claimTestBus) EmitReleaseNew(context.Context, events.ReleaseEvent)      {}
func (claimTestBus) EmitReleasePush(context.Context, events.ReleasePushEvent) {}

type claimTestCall struct {
	indexer  string
	filterID int
	action   string
	approved bool
}

// claimTestActions uses the real action repo for lookups and stands in for the download client.
type claimTestActions struct {
	repo *database.ActionRepo

	hold     time.Duration
	entered  chan string
	prepare  func(action *domain.Action, release *domain.Release) error
	run      func(action *domain.Action, release *domain.Release) []string
	panicked atomic.Bool

	mu    sync.Mutex
	calls []claimTestCall
}

func (a *claimTestActions) Get(context.Context, *domain.GetActionRequest) (*domain.Action, error) {
	return nil, errors.New("not used")
}

func (a *claimTestActions) FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*domain.Action, error) {
	return a.repo.FindByFilterID(ctx, filterID, active, withClient)
}

func (a *claimTestActions) CheckActionPreconditions(_ context.Context, action *domain.Action, release *domain.Release) error {
	if a.prepare != nil {
		return a.prepare(action, release)
	}

	return nil
}

// RunAction models a client push: the torrent lands in the client, then the re-announce loop
// holds the push in flight before a verdict comes back.
func (a *claimTestActions) RunAction(_ context.Context, action *domain.Action, release *domain.Release) ([]string, error) {
	if a.entered != nil {
		a.entered <- release.Indexer.Identifier
	}

	time.Sleep(a.hold)

	var rejections []string
	if a.run != nil {
		rejections = a.run(action, release)
	}

	a.mu.Lock()
	a.calls = append(a.calls, claimTestCall{indexer: release.Indexer.Identifier, filterID: release.FilterID, action: action.Name, approved: len(rejections) == 0})
	a.mu.Unlock()

	return rejections, nil
}

func (a *claimTestActions) approved() []claimTestCall {
	a.mu.Lock()
	defer a.mu.Unlock()

	var out []claimTestCall
	for _, c := range a.calls {
		if c.approved {
			out = append(out, c)
		}
	}

	return out
}

func (a *claimTestActions) ran() []string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var out []string
	for _, c := range a.calls {
		out = append(out, c.action)
	}

	return out
}

type claimTestEnv struct {
	db         *database.DB
	svc        *Service
	actions    *claimTestActions
	actionRepo *database.ActionRepo
	filterRepo *database.FilterRepo
	indexers   map[string]domain.IndexerMinimal
}

func newClaimTestEnv(t *testing.T, engine string, indexers ...string) *claimTestEnv {
	t.Helper()

	ctx := t.Context()
	log := logger.Mock()
	db := openClaimTestDB(t, engine)

	releaseRepo := database.NewReleaseRepo(log, db)
	filterRepo := database.NewFilterRepo(log, db)
	actionRepo := database.NewActionRepo(log, db)
	indexerRepo := database.NewIndexerRepo(log, db)

	env := &claimTestEnv{
		db:         db,
		actionRepo: actionRepo,
		filterRepo: filterRepo,
		actions:    &claimTestActions{repo: actionRepo},
		indexers:   map[string]domain.IndexerMinimal{},
	}

	for _, identifier := range indexers {
		idx, err := indexerRepo.Store(ctx, domain.Indexer{Name: identifier, Identifier: identifier, Enabled: true, Implementation: domain.IndexerImplementationIRC})
		require.NoError(t, err)
		env.indexers[identifier] = domain.IndexerMinimal{ID: int(idx.ID), Name: idx.Name, Identifier: idx.Identifier}
	}

	filterSvc := filter.NewService(log, filterRepo, nil, releaseRepo, nil, nil, nil, nil)
	env.svc = NewService(log, claimTestBus{}, releaseRepo, env.actions, filterSvc, nil, nil)

	return env
}

type claimTestFilter struct {
	name      string
	priority  int32
	profileID int64
	indexers  []string
	actions   []domain.ActionType
	configure func(f *domain.Filter)
}

func (e *claimTestEnv) addFilter(t *testing.T, cf claimTestFilter) *domain.Filter {
	t.Helper()
	ctx := t.Context()

	f := &domain.Filter{Name: cf.name, Enabled: true, Priority: cf.priority, Resolutions: []string{}, Codecs: []string{}, Sources: []string{}, Containers: []string{}}
	if cf.profileID > 0 {
		f.ReleaseProfileDuplicateID = cf.profileID
	}
	if cf.configure != nil {
		cf.configure(f)
	}
	require.NoError(t, e.filterRepo.Store(ctx, f))

	indexers := cf.indexers
	if len(indexers) == 0 {
		for identifier := range e.indexers {
			indexers = append(indexers, identifier)
		}
	}
	for _, identifier := range indexers {
		require.NoError(t, e.filterRepo.StoreIndexerConnection(ctx, f.ID, e.indexers[identifier].ID))
	}

	for i, typ := range cf.actions {
		act := &domain.Action{Name: fmt.Sprintf("%s-action-%d", cf.name, i), Type: typ, Enabled: true, FilterID: f.ID}
		require.NoError(t, e.actionRepo.Store(ctx, act))
	}

	return f
}

func (e *claimTestEnv) newRelease(indexer, name string) *domain.Release {
	rls := domain.NewRelease(e.indexers[indexer])
	rls.ParseString(name)

	return rls
}

type claimTestStatus struct {
	action     string
	status     domain.ReleasePushStatus
	rejections []string
}

func (e *claimTestEnv) statuses(t *testing.T) []claimTestStatus {
	t.Helper()

	rows, err := e.db.Handler.QueryContext(t.Context(), `SELECT action, status, rejections FROM release_action_status ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	var out []claimTestStatus
	for rows.Next() {
		var s claimTestStatus
		require.NoError(t, rows.Scan(&s.action, &s.status, pq.Array(&s.rejections)))
		out = append(out, s)
	}
	require.NoError(t, rows.Err())

	return out
}

func (e *claimTestEnv) countStatus(t *testing.T, status domain.ReleasePushStatus) int {
	t.Helper()

	var n int
	require.NoError(t, e.db.Handler.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM release_action_status WHERE status = $1`, string(status)).Scan(&n))

	return n
}

// processAtOnce processes the releases from parallel goroutines released by one barrier.
func (e *claimTestEnv) processAtOnce(releases ...*domain.Release) {
	start := make(chan struct{})

	var wg sync.WaitGroup
	for _, rls := range releases {
		wg.Go(func() {
			<-start
			e.svc.Process(context.Background(), rls)
		})
	}

	close(start)
	wg.Wait()
}

const claimTestReleaseName = "That.Movie.2023.1080p.BluRay.x264-GROUP"

// #1845: three trackers announce the same release at the same moment and the filter skips
// duplicates, so exactly one copy may be pushed.
func TestClaim_ConcurrentAnnouncesPushOnce(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a", "tracker-b", "tracker-c")
		env.actions.hold = 20 * time.Millisecond
		env.addFilter(t, claimTestFilter{name: "movies", priority: 1, profileID: claimExactProfileID, actions: []domain.ActionType{domain.ActionTypeTest}})

		const rounds = 10
		for round := range rounds {
			name := fmt.Sprintf("That.Movie.%d.1080p.BluRay.x264-GROUP", 2000+round)
			env.processAtOnce(env.newRelease("tracker-a", name), env.newRelease("tracker-b", name), env.newRelease("tracker-c", name))
		}

		assert.Len(t, env.actions.approved(), rounds, "one push per release")
		assert.Zero(t, env.countStatus(t, domain.ReleasePushStatusPending))
	})
}

// #2503: the second copy arrives while the first push is still in flight. It is rejected as a
// duplicate of a release being pushed and carries the in-flight marker for the feed re-check.
func TestClaim_SecondAnnounceDuringFirstPush(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-irc", "tracker-rss")
		env.actions.entered = make(chan string, 1)
		env.addFilter(t, claimTestFilter{name: "irc", priority: 2, indexers: []string{"tracker-irc"}, actions: []domain.ActionType{domain.ActionTypeTest}})
		env.addFilter(t, claimTestFilter{name: "rss", priority: 1, profileID: claimExactProfileID, indexers: []string{"tracker-rss"}, actions: []domain.ActionType{domain.ActionTypeTest}})

		release := make(chan struct{})
		env.actions.run = func(*domain.Action, *domain.Release) []string {
			<-release
			return nil
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			env.svc.Process(context.Background(), env.newRelease("tracker-irc", claimTestReleaseName))
		}()

		require.Equal(t, "tracker-irc", <-env.actions.entered)

		second := env.newRelease("tracker-rss", claimTestReleaseName)
		env.svc.Process(context.Background(), second)

		close(release)
		<-done

		assert.Len(t, env.actions.approved(), 1)
		assert.True(t, second.IsDuplicate)
		assert.True(t, second.DuplicateInFlight)
	})
}

// A copy rejected because the matching push already went through is a plain duplicate, so the
// feed must not check it again.
func TestClaim_DuplicateOfApprovedIsNotInFlight(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a", "tracker-b")
		env.addFilter(t, claimTestFilter{name: "movies", priority: 1, profileID: claimExactProfileID, actions: []domain.ActionType{domain.ActionTypeTest}})

		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))

		second := env.newRelease("tracker-b", claimTestReleaseName)
		env.svc.Process(context.Background(), second)

		assert.Len(t, env.actions.approved(), 1)
		assert.True(t, second.IsDuplicate)
		assert.False(t, second.DuplicateInFlight)
	})
}

// The #2503 producer: an IRC filter without a profile and an RSS backup filter with Exact
// release. The RSS copy passes its optimistic check before the IRC push claims, and claims right
// after it. The filter without a profile must claim too, or the backup can't see its push.
func TestClaim_FilterWithoutProfileClaims(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-irc", "tracker-rss")
		env.actions.entered = make(chan string, 1)
		env.addFilter(t, claimTestFilter{name: "irc", priority: 2, indexers: []string{"tracker-irc"}, actions: []domain.ActionType{domain.ActionTypeTest}})
		env.addFilter(t, claimTestFilter{name: "rss", priority: 1, profileID: claimExactProfileID, indexers: []string{"tracker-rss"}, actions: []domain.ActionType{domain.ActionTypeTest}})

		rssChecked := make(chan struct{})
		ircClaimed := make(chan struct{})
		env.actions.prepare = func(_ *domain.Action, rls *domain.Release) error {
			if rls.Indexer.Identifier == "tracker-irc" {
				<-rssChecked
				return nil
			}

			close(rssChecked)
			<-ircClaimed

			return nil
		}

		backupDone := make(chan struct{})
		env.actions.run = func(_ *domain.Action, rls *domain.Release) []string {
			if rls.Indexer.Identifier == "tracker-irc" {
				<-backupDone
			}
			return nil
		}

		ircDone := make(chan struct{})
		go func() {
			defer close(ircDone)
			env.svc.Process(context.Background(), env.newRelease("tracker-irc", claimTestReleaseName))
		}()

		backup := env.newRelease("tracker-rss", claimTestReleaseName)
		go func() {
			defer close(backupDone)
			env.svc.Process(context.Background(), backup)
		}()

		require.Equal(t, "tracker-irc", <-env.actions.entered)
		close(ircClaimed)
		<-backupDone
		<-ircDone

		approved := env.actions.approved()
		require.Len(t, approved, 1, "the backup must not push a release the primary claimed")
		assert.Equal(t, "tracker-irc", approved[0].indexer)
		assert.True(t, backup.DuplicateInFlight)
	})
}

// Max downloads counts claims, so concurrent releases can't overshoot the limit.
func TestClaim_MaxDownloadsConcurrent(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a")
		env.actions.hold = 20 * time.Millisecond
		env.addFilter(t, claimTestFilter{name: "limited", priority: 1, actions: []domain.ActionType{domain.ActionTypeTest}, configure: func(f *domain.Filter) {
			f.MaxDownloads = 2
			f.MaxDownloadsUnit = domain.FilterMaxDownloadsDay
		}})

		var releases []*domain.Release
		for i := range 5 {
			releases = append(releases, env.newRelease("tracker-a", fmt.Sprintf("Other.Movie.%d.1080p.BluRay.x264-GROUP", 2000+i)))
		}
		env.processAtOnce(releases...)

		assert.Len(t, env.actions.approved(), 2)
	})
}

// An action skipped because its client already rejected the release leaves no row behind, and
// a filter whose every client was tried hands over to the next filter.
func TestClaim_TriedClientLeavesNoRow(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a")
		env.addFilter(t, claimTestFilter{name: "c-hi", priority: 3, actions: []domain.ActionType{domain.ActionTypeTest}})
		env.addFilter(t, claimTestFilter{name: "b-mid", priority: 2, actions: []domain.ActionType{domain.ActionTypeTest}})
		env.addFilter(t, claimTestFilter{name: "a-lo", priority: 1, actions: []domain.ActionType{domain.ActionTypeWebhook}})

		env.actions.run = func(action *domain.Action, _ *domain.Release) []string {
			if action.Name == "c-hi-action-0" {
				return []string{"max active downloads reached, skipping"}
			}
			return nil
		}

		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))

		assert.Equal(t, []string{"c-hi-action-0", "a-lo-action-0"}, env.actions.ran())
		assert.Equal(t, []claimTestStatus{
			{action: "c-hi-action-0", status: domain.ReleasePushStatusRejected, rejections: []string{"max active downloads reached, skipping"}},
			{action: "a-lo-action-0", status: domain.ReleasePushStatusApproved, rejections: []string{}},
		}, env.statuses(t))
	})
}

// A filter with one approved and one rejected action pushed the release, so later filters
// don't run.
func TestClaim_PartialSuccessStopsFilterLoop(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a")
		env.addFilter(t, claimTestFilter{name: "hi", priority: 2, actions: []domain.ActionType{domain.ActionTypeTest, domain.ActionTypeWebhook}})
		env.addFilter(t, claimTestFilter{name: "lo", priority: 1, actions: []domain.ActionType{domain.ActionTypeExec}})

		env.actions.run = func(action *domain.Action, _ *domain.Release) []string {
			if action.Name == "hi-action-1" {
				return []string{"rejected by arr"}
			}
			return nil
		}

		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))

		assert.Equal(t, []string{"hi-action-0", "hi-action-1"}, env.actions.ran())
	})
}

// A panicking action can't leave its claim PENDING, so the next copy is not blocked forever.
func TestClaim_PanicResolvesPending(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a", "tracker-b")
		env.addFilter(t, claimTestFilter{name: "movies", priority: 1, profileID: claimExactProfileID, actions: []domain.ActionType{domain.ActionTypeTest}})

		env.actions.run = func(*domain.Action, *domain.Release) []string {
			if env.actions.panicked.CompareAndSwap(false, true) {
				panic("client exploded")
			}
			return nil
		}

		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))

		assert.Equal(t, []claimTestStatus{
			{action: "movies-action-0", status: domain.ReleasePushStatusErr, rejections: []string{"action did not finish"}},
		}, env.statuses(t))

		env.svc.Process(context.Background(), env.newRelease("tracker-b", claimTestReleaseName))

		assert.Len(t, env.actions.approved(), 1, "the second copy is pushed once the first failed")
	})
}

// A torrent that can't be downloaded is recorded as a failed push without taking a claim, and the
// next filter with another client still gets the release.
func TestClaim_PrepareErrorFailsFilter(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a")
		env.addFilter(t, claimTestFilter{name: "hi", priority: 2, actions: []domain.ActionType{domain.ActionTypeQbittorrent}})
		env.addFilter(t, claimTestFilter{name: "lo", priority: 1, actions: []domain.ActionType{domain.ActionTypeExec}})

		env.actions.prepare = func(action *domain.Action, _ *domain.Release) error {
			if action.Type == domain.ActionTypeQbittorrent {
				return errors.New("could not download torrent file: 404")
			}
			return nil
		}

		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))

		assert.Equal(t, []string{"lo-action-0"}, env.actions.ran())
		assert.Equal(t, []claimTestStatus{
			{action: "hi-action-0", status: domain.ReleasePushStatusErr, rejections: []string{"could not download torrent file: 404"}},
			{action: "lo-action-0", status: domain.ReleasePushStatusApproved, rejections: []string{}},
		}, env.statuses(t))
	})
}

// A filter delay sits before the claim, so a delayed filter that matched first doesn't hold the
// release while a filter without delay on another tracker pushes it.
func TestClaim_DelayDoesNotBlockOtherTracker(t *testing.T) {
	forEachEngine(t, func(t *testing.T, engine string) {
		env := newClaimTestEnv(t, engine, "tracker-a", "tracker-b")
		env.addFilter(t, claimTestFilter{name: "slow", priority: 1, profileID: claimExactProfileID, indexers: []string{"tracker-b"}, actions: []domain.ActionType{domain.ActionTypeTest}, configure: func(f *domain.Filter) {
			f.Delay = 1
		}})
		env.addFilter(t, claimTestFilter{name: "fast", priority: 1, profileID: claimExactProfileID, indexers: []string{"tracker-a"}, actions: []domain.ActionType{domain.ActionTypeTest}})

		slowDone := make(chan struct{})
		go func() {
			defer close(slowDone)
			env.svc.Process(context.Background(), env.newRelease("tracker-b", claimTestReleaseName))
		}()

		time.Sleep(50 * time.Millisecond)

		started := time.Now()
		env.svc.Process(context.Background(), env.newRelease("tracker-a", claimTestReleaseName))
		assert.Less(t, time.Since(started), 500*time.Millisecond, "the fast tracker must not wait for the delayed one")

		<-slowDone

		approved := env.actions.approved()
		require.Len(t, approved, 1)
		assert.Equal(t, "tracker-a", approved[0].indexer)
	})
}
