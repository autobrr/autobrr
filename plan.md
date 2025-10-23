# Migration Plan: Config → Database-Based Release Cleanup Jobs

## Executive Summary
Transform release cleanup from single config.toml-based job to **multiple database-backed cleanup jobs** with full CRUD API support. Follows the **Feed pattern** architecture. **No backwards compatibility needed** - cleanup functionality hasn't been released yet.

**UI Status:** Not included in this plan. Full REST API provided allows UI to be added later without any backend code changes.

---

## ✅ STAGE 1: Database Foundation (COMPLETE)

### What Was Built

**Files Created:**
1. `internal/database/migrations/sqlite/84_create_release_cleanup_job.sql` - SQLite migration
2. `internal/database/migrations/postgres/74_create_release_cleanup_job.sql` - Postgres migration
3. `internal/database/release_cleanup_job.go` - Repository implementation (329 lines)
4. `internal/database/release_cleanup_job_test.go` - Integration tests (308 lines)

**Files Modified:**
1. `internal/domain/release.go` - Added ReleaseCleanupJob struct, ReleaseCleanupStatus enum, ReleaseCleanupJobRepo interface
2. `internal/database/migrations/sqlite.go` - Registered migration 84 (auto-generated via go generate)
3. `internal/database/migrations/postgres.go` - Registered migration 74 (manual)
4. `internal/database/migrations/sqlite/current_schema_sqlite.sql` - Added table definition
5. `internal/database/migrations/postgres/current_schema_postgres.sql` - Added table definition

### Database Schema

**Table:** `release_cleanup_job` (no UNIQUE constraint on name - follows feed/list/notification pattern)

**Columns:**
- `id` - Primary key
- `name` - TEXT NOT NULL (user-friendly label)
- `enabled` - BOOLEAN DEFAULT FALSE
- `schedule` - TEXT NOT NULL (cron format: "0 3 * * *")
- `older_than` - INTEGER NOT NULL (hours: 720 = 30 days)
- `indexers` - TEXT (comma-separated: "btn,ptp" or NULL for all)
- `statuses` - TEXT (comma-separated: "PUSH_REJECTED,PUSH_ERROR" or NULL for all)
- `last_run` - TIMESTAMP (NULL until first run)
- `last_run_status` - TEXT ("SUCCESS" or "ERROR")
- `last_run_data` - TEXT (JSON stats or error message)
- `created_at` - TIMESTAMP DEFAULT CURRENT_TIMESTAMP
- `updated_at` - TIMESTAMP DEFAULT CURRENT_TIMESTAMP

**No indexes added** - enabled index was removed to match feed pattern (small table, low cardinality)

### Repository Implementation

**Pattern:** Follows `internal/database/feed.go` exactly
- Squirrel query builder for all queries
- NULL handling with sql.NullString/sql.NullTime
- RETURNING id on INSERT
- RowsAffected check on UPDATE/DELETE
- domain.ErrRecordNotFound for not found cases

**Methods (all implemented):**
- `List()` - ORDER BY name ASC, returns all jobs
- `FindByID()` - Single job lookup
- `Store()` - INSERT with RETURNING id
- `Update()` - UPDATE all core fields, SET updated_at
- `UpdateLastRun()` - UPDATE only last_run/status/data (for job execution tracking)
- `ToggleEnabled()` - UPDATE enabled field, SET updated_at
- `Delete()` - DELETE with rowsAffected check

### Test Coverage

**File:** `internal/database/release_cleanup_job_test.go`

**7 test functions, 14 scenarios (both SQLite and Postgres):**
- Store_Succeeds
- FindByID_Succeeds + FindByID_Fails_Not_Found
- List_Returns_All_Jobs (3 jobs) + List_Empty_Table
- Update_Succeeds + Update_Fails_Non_Existing_Job
- UpdateLastRun_Succeeds + UpdateLastRun_Fails_Non_Existing_Job
- ToggleEnabled_Succeeds (both directions) + ToggleEnabled_Fails_Non_Existing_Job
- Delete_Succeeds (with verification) + Delete_Fails_Non_Existing_Job

**Test Pattern:** Follows `internal/database/feed_test.go` exactly
- Uses `assert.ErrorIs(err, domain.ErrRecordNotFound)` (newer pattern than feed_test.go)
- Mock data factory: `getMockReleaseCleanupJob()`
- Integration tests with real databases (not mocks)

### Running Tests

```bash
# Full integration test suite (both SQLite + Postgres)
go test ./internal/database -tags=integration -v

# Just cleanup job tests
go test ./internal/database -tags=integration -run TestReleaseCleanupJobRepo -v

# Clean cache first
go clean -testcache
```

**Database Requirements:**
- Postgres container must be running on `localhost:5437`
- Check: `docker ps | grep postgres`
- SQLite runs in-memory (no external dependency)

### Key Implementation Details

**Migration Registration:**
- SQLite: Auto-generated via `go generate ./internal/database/migrations/codegen`
- Postgres: Manually added to `internal/database/migrations/postgres.go`

**Base Schema Files:**
- Both `current_schema_*.sql` files updated with table definition
- Required for integration tests (base schema applied when migration count = 0)
- **Critical:** Without base schema update, tests fail with "relation does not exist"

**NULL Field Handling:**
- Optional fields (indexers, statuses, last_run*) use sql.NullString/sql.NullTime
- Empty strings converted to NULL on Store/Update
- NULL converted back to empty strings on Scan

**Error Handling:**
- All methods return `domain.ErrRecordNotFound` for missing records
- Wraps all errors with context: `errors.Wrap(err, "error building query")`
- Follows established error handling pattern across codebase

### Gotchas & Lessons Learned

1. **Base schema files must be updated** - Integration tests apply base schema on clean DB, not incremental migrations
2. **Go embed is compile-time** - Must rebuild tests after creating new migration files (`go clean -testcache`)
3. **Postgres migrations are manual** - Only SQLite has go generate script
4. **No enabled index needed** - Small tables with boolean columns don't benefit from indexes
5. **Name uniqueness not enforced** - Follows feed/list/notification pattern (UI problem, not DB constraint)

---

## ✅ STAGE 2: Service Layer (COMPLETE)

### What Was Built

**Files Modified:**
1. `internal/release/cleanup.go` - Refactored to use `*domain.ReleaseCleanupJob`, added status tracking
2. `internal/release/service.go` - Added CRUD methods, lifecycle methods, updated Start()
3. `internal/domain/release.go` - Added NextRun field to ReleaseCleanupJob
4. `cmd/autobrr/main.go` - Wired up releaseCleanupJobRepo dependency

**Files Created:**
1. `internal/release/service_test.go` - Basic service tests for cleanupJobKey

### CleanupJob Refactoring

**Constructor Changes:**
- **Old:** `NewCleanupJob(log, releaseRepo, config)`
- **New:** `NewCleanupJob(log, releaseRepo, cleanupJobRepo, job)`

**Status Tracking:**
- Sets `LastRun` timestamp on execution start
- On success: Updates `LastRunStatus` = SUCCESS, `LastRunData` = JSON with stats
- On error: Updates `LastRunStatus` = ERROR, `LastRunData` = error message
- Uses `cleanupJobRepo.UpdateLastRun()` to persist status

### Service Layer Implementation

**New Fields:**
- `cleanupJobs map[string]int` - Tracks scheduler job IDs by cleanup job ID
- `cleanupJobRepo domain.ReleaseCleanupJobRepo` - Repository for cleanup jobs

**Job Identifier:**
- Type: `cleanupJobKey{id int}`
- Format: `"release-cleanup-{id}"` (e.g., "release-cleanup-42")

**CRUD Methods (6 methods):**
1. `ListCleanupJobs(ctx)` - Returns all jobs with NextRun enriched from scheduler
2. `GetCleanupJob(ctx, id)` - Returns single job by ID
3. `StoreCleanupJob(ctx, job)` - Creates job, starts if enabled
4. `UpdateCleanupJob(ctx, job)` - Updates job, restarts to pick up changes
5. `DeleteCleanupJob(ctx, id)` - Stops job, deletes from database
6. `ToggleCleanupJobEnabled(ctx, id, enabled)` - Starts or stops based on enabled flag

**Lifecycle Methods (3 private methods):**
1. `startCleanupJob(job)` - Creates CleanupJob instance, schedules with cron, adds to jobs map
2. `stopCleanupJob(id)` - Removes from scheduler, deletes from jobs map
3. `restartCleanupJob(job)` - Stops then starts if enabled

**Start() Implementation:**
- Loads all cleanup jobs from database via `cleanupJobRepo.List()`
- Starts enabled jobs in background goroutine
- Staggered start with 2-second sleep between jobs
- Logs job count and any failures

### CRUD → Lifecycle Wiring

- **Store:** Saves to DB → `startCleanupJob()` if enabled
- **Update:** Updates DB → `restartCleanupJob()` to pick up changes
- **Delete:** `stopCleanupJob()` → Deletes from DB
- **ToggleEnabled:** Updates DB → `startCleanupJob()` or `stopCleanupJob()`

### Pattern Compliance

Followed `internal/feed/service.go` exactly:
- Job map tracking with string keys
- List() enriches NextRun from scheduler
- Update → restart pattern
- Delete → stop then delete pattern
- Start() loads from DB in background goroutine
- Staggered startup with sleep

### Testing

**File:** `internal/release/service_test.go`
- Tests cleanupJobKey.ToString() with multiple IDs
- Comprehensive service testing deferred to integration/API layer tests

### Build Status

✅ Full project builds successfully with `go build ./...`

---

## 📋 STAGE 3: API Layer (NEXT)

### Goals
- Add REST endpoints to `internal/http/release.go`
- Add releaseService interface methods
- Add HTTP handlers for all CRUD operations

### Routes
```
GET    /api/releases/cleanup-jobs      - List all jobs
POST   /api/releases/cleanup-jobs      - Create job
GET    /api/releases/cleanup-jobs/:id  - Get job
PUT    /api/releases/cleanup-jobs/:id  - Update job
DELETE /api/releases/cleanup-jobs/:id  - Delete job
PATCH  /api/releases/cleanup-jobs/:id/enabled - Toggle enabled
```

### Pattern Reference
Follow `internal/http/feed.go` exactly

---

## 🧹 STAGE 4: Cleanup & Integration

### Remove Config Approach
**Files to modify:**
1. `internal/domain/config.go` - Remove 5 cleanup fields
2. `internal/config/config.go` - Remove defaults, env loading, template
3. `README.md` - Remove 5 environment variable rows

### Wire Up Dependencies
**File:** `cmd/autobrr/main.go`
- Initialize: `releaseCleanupJobRepo := database.NewReleaseCleanupJobRepo(log, db)`
- Update: `release.NewService()` constructor to include cleanupJobRepo

---

## 📖 STAGE 5: Documentation

- API endpoint documentation
- Database schema documentation
- Cron schedule examples
- Migration notes

---

## 🎨 STAGE 6: UI Implementation (Future)

**Status:** Not in scope - full REST API ready for future UI consumption

---

## Testing Strategy

### Unit Tests
- Repository: CRUD on both SQLite and Postgres ✅
- Service: CRUD methods, job lifecycle
- HTTP: All 6 API endpoints

### Integration Tests
- Create job via API → Scheduled in cron
- Update job → Reschedules
- Toggle enabled → Starts/stops
- Delete job → Removes from scheduler
- Service.Start() loads enabled jobs
- Job execution updates status

### Manual Testing
- Migrations run cleanly
- Multiple jobs run independently
- Status tracking works (SUCCESS/ERROR)
- Cron schedules trigger correctly

---

## Current Branch Status

**Branch:** `feat/scheduled-release-cleanup`

**Commits on branch:**
1. `feat(release): add scheduled cleanup with configurable retention` (old config approach)
2. `docs(readme): add release cleanup environment variables` (old config approach)
3. `test(database): add comprehensive Delete test suite` (kept - still needed)

**Stage 1 commits will supersede commits 1-2** when we complete and commit all stages.
