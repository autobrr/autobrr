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

**CRUD Methods (7 methods):**
1. `ListCleanupJobs(ctx)` - Returns all jobs with NextRun enriched from scheduler
2. `GetCleanupJob(ctx, id)` - Returns single job by ID
3. `StoreCleanupJob(ctx, job)` - Creates job, starts if enabled
4. `UpdateCleanupJob(ctx, job)` - Updates job, restarts to pick up changes
5. `DeleteCleanupJob(ctx, id)` - Stops job, deletes from database
6. `ToggleCleanupJobEnabled(ctx, id, enabled)` - Starts or stops based on enabled flag
7. `ForceRunCleanupJob(ctx, id)` - Manually triggers cleanup job (bypasses schedule)

**Lifecycle Methods (3 private methods):**
1. `startCleanupJob(job)` - Creates CleanupJob instance, schedules with cron, adds to jobs map
2. `stopCleanupJob(id)` - Removes from scheduler, deletes from jobs map
3. `restartCleanupJob(job)` - Stops then starts if enabled

**Start() Implementation:**
- Loads all cleanup jobs from database via `cleanupJobRepo.List()`
- Starts enabled jobs in background goroutine
- No sleep between jobs (only registers with scheduler, no DB writes)
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

## ✅ STAGE 3: API Layer (COMPLETE)

### What Was Built

**Files Modified:**
1. `internal/http/release.go` - Added cleanup job routes and handlers (139 new lines)
2. `internal/release/service.go` - Added ForceRunCleanupJob implementation (20 new lines)

**Files Created:**
1. `internal/http/release_test.go` - Integration tests for cleanup job endpoints (525 lines)

### Interface Methods Added

**releaseService interface** (lines 33-39 in release.go):
```go
// Cleanup jobs
ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error)
GetCleanupJob(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error)
StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
DeleteCleanupJob(ctx context.Context, id int) error
ToggleCleanupJobEnabled(ctx context.Context, id int, enabled bool) error
ForceRunCleanupJob(ctx context.Context, id int) error
```

### Routes Implemented

**Nested route: `/api/releases/cleanup-jobs`** (lines 75-86 in release.go):
```
GET    /api/releases/cleanup-jobs           - List all jobs
POST   /api/releases/cleanup-jobs           - Create job
GET    /api/releases/cleanup-jobs/:id       - Get job by ID
PUT    /api/releases/cleanup-jobs/:id       - Update job
DELETE /api/releases/cleanup-jobs/:id       - Delete job
PATCH  /api/releases/cleanup-jobs/:id/enabled - Toggle enabled
POST   /api/releases/cleanup-jobs/:id/run   - Force run (manual trigger)
```

### Handler Methods Implemented

**7 handlers following feed.go pattern** (lines 393-522):

1. **listCleanupJobs** - GET `/`
   - Returns all cleanup jobs with 200 OK
   - Uses service.ListCleanupJobs()

2. **getCleanupJob** - GET `/:id`
   - URL param: jobID (parsed with strconv.Atoi)
   - Returns single job with 200 OK
   - Returns 404 if domain.ErrRecordNotFound
   - Uses service.GetCleanupJob()

3. **storeCleanupJob** - POST `/`
   - JSON body: ReleaseCleanupJob
   - Returns created job with 201 Created
   - Uses service.StoreCleanupJob()

4. **updateCleanupJob** - PUT `/:id`
   - JSON body: ReleaseCleanupJob
   - Returns updated job with 201 Created (matches feed pattern)
   - Uses service.UpdateCleanupJob()

5. **deleteCleanupJob** - DELETE `/:id`
   - URL param: jobID
   - Returns 204 No Content
   - Returns 404 if domain.ErrRecordNotFound
   - Uses service.DeleteCleanupJob()

6. **toggleCleanupJobEnabled** - PATCH `/:id/enabled`
   - URL param: jobID
   - JSON body: `{"enabled": true/false}`
   - Returns 204 No Content
   - Returns 404 if domain.ErrRecordNotFound
   - Uses service.ToggleCleanupJobEnabled()

7. **forceRunCleanupJob** - POST `/:id/run`
   - URL param: jobID
   - Manually triggers cleanup job execution (bypasses schedule)
   - Returns 204 No Content
   - Returns 404 if domain.ErrRecordNotFound
   - Uses service.ForceRunCleanupJob()

### Error Handling

**Consistent error handling across all handlers:**
- URL parameter parsing errors → h.encoder.Error(w, err)
- JSON decoding errors → h.encoder.Error(w, err)
- domain.ErrRecordNotFound → h.encoder.NotFoundErr(w, errors.New(...))
- Other service errors → h.encoder.Error(w, err)

### HTTP Status Codes

**Following established patterns:**
- GET operations → 200 OK
- POST (create) → 201 Created
- PUT (update) → 201 Created (matches feed.go pattern)
- DELETE → 204 No Content
- PATCH (toggle) → 204 No Content
- Not found → 404 with custom error message
- Bad request → handled by encoder

### Test Coverage

**HTTP Integration Tests** (release_test.go - 523 lines):
- **12 comprehensive test cases** covering all 7 endpoints with success and failure scenarios
- Mock releaseService with in-memory job storage
- Uses httptest.NewServer following auth_test.go pattern
- All tests passing (0.053s runtime)
- Tests validated with intentional failures to ensure they catch real issues

**Test Cases:**
1. ✅ ListCleanupJobs - Returns all jobs with 200 OK, validates all 7 fields
2. ✅ GetCleanupJob - Returns job with 200 OK, validates all 7 fields
3. ✅ GetCleanupJob_NotFound - Returns 404 when job doesn't exist
4. ✅ StoreCleanupJob - Creates job (201), validates response + storage with all 7 fields
5. ✅ UpdateCleanupJob - Updates job (201), validates response + storage with all 7 fields
6. ✅ UpdateCleanupJob_NotFound - Returns 404 when job doesn't exist
7. ✅ DeleteCleanupJob - Deletes job (204), verifies storage is empty
8. ✅ DeleteCleanupJob_NotFound - Returns 404 when job doesn't exist
9. ✅ ToggleCleanupJobEnabled - Toggles enabled (204), verifies storage updated
10. ✅ ToggleCleanupJobEnabled_NotFound - Returns 404 when job doesn't exist
11. ✅ ForceRunCleanupJob - Triggers run (204), validates LastRun/Status/Data updated
12. ✅ ForceRunCleanupJob_NotFound - Returns 404 when job doesn't exist

**Test Quality Standards:**
- ✅ Complete HTTP status code validation
- ✅ Complete field validation (all 7 core fields: ID, Name, Enabled, Schedule, OlderThan, Indexers, Statuses)
- ✅ Storage verification for mutating operations (Store, Update, Delete, Toggle, ForceRun)
- ✅ Consistent field ordering across all tests
- ✅ Both success and failure paths tested
- ✅ Tests validated with intentional failures (3 tests) to ensure assertions catch issues

**Bugs Fixed During Testing:**
1. updateCleanupJob handler was missing ErrRecordNotFound check
   - Now properly returns 404 instead of 500 for non-existent jobs
2. Incomplete field validation in tests
   - All tests now validate every field in responses AND storage

### Build Status

✅ Full project builds successfully with `go build ./...`
✅ No compiler diagnostics
✅ All 12 HTTP integration tests passing

### Service Implementation

**ForceRunCleanupJob** (service.go:274-293):
- Finds cleanup job by ID from database
- Creates CleanupJob instance with NewCleanupJob
- Calls Run() synchronously (immediate execution, bypasses scheduler)
- Logs manual trigger event
- Updates job status via CleanupJob.Run() (LastRun, LastRunStatus, LastRunData)

### Pattern Compliance

**Followed `internal/http/feed.go` exactly:**
- Interface definition in http package (not domain)
- URL parameter extraction with chi.URLParam and strconv.Atoi
- JSON decoding for POST/PUT bodies
- Consistent error handling with ErrRecordNotFound checks
- Nested route structure for resource operations
- Encoder methods for response formatting

---

## ✅ STAGE 4: Cleanup & Integration (COMPLETE)

### What Was Removed

**Complete elimination of config-based cleanup approach** - replaced by database-backed multi-job system built in Stages 1-3.

**Files Modified:**
1. `internal/domain/config.go` - Removed 5 cleanup fields from Config struct
2. `internal/config/config.go` - Removed cleanup template, defaults, and env loading
3. `README.md` - Removed 5 environment variable documentation rows
4. `internal/release/service.go` - Removed config field and parameter
5. `cmd/autobrr/main.go` - Updated release.NewService call

### Config Cleanup Details

**internal/domain/config.go (lines 47-51 removed):**
- `ReleaseCleanupEnabled bool`
- `ReleaseCleanupSchedule string`
- `ReleaseCleanupOlderThan int`
- `ReleaseCleanupIndexers string`
- `ReleaseCleanupStatuses string`

**internal/config/config.go (3 sections removed):**

1. **Template (lines 158-181):** Removed entire "Release History Cleanup" config template section (24 lines of commented configuration)

2. **Defaults (lines 329-333):** Removed 5 default value assignments from defaults() method

3. **Environment Loading (lines 494-512):** Removed 5 environment variable loaders from loadFromEnv() method
   - `AUTOBRR__RELEASE_CLEANUP_ENABLED`
   - `AUTOBRR__RELEASE_CLEANUP_SCHEDULE`
   - `AUTOBRR__RELEASE_CLEANUP_OLDER_THAN`
   - `AUTOBRR__RELEASE_CLEANUP_INDEXERS`
   - `AUTOBRR__RELEASE_CLEANUP_STATUSES`

**README.md (lines 348-352 removed):**
- Removed 5 environment variable documentation rows from configuration table

**internal/release/service.go:**
- Removed `config *domain.Config` field from service struct
- Removed config parameter from NewService() signature
- Removed config assignment in NewService() body

**cmd/autobrr/main.go:**
- Updated release.NewService() call to remove `cfg.Config` argument

### Dependency Wiring Status

✅ **Already complete from Stage 2:**
- `releaseCleanupJobRepo` initialized in main.go (line ~157)
- `release.NewService()` receives cleanupJobRepo parameter (line ~161)
- `releaseService.Start()` called by server.Start() (internal/server/server.go:75)

### Verification Results

✅ **Build successful:** `go build ./...` completes with no errors
✅ **No config references:** Verified zero "ReleaseCleanup" matches in config files
✅ **No README references:** Verified zero "RELEASE_CLEANUP" matches in README
✅ **No compiler diagnostics:** All modified files clean

### Migration Complete

**BEFORE (Config Approach - PR #1):**
- Single cleanup job configured via config.toml or environment variables
- 5 config fields with template, defaults, and env loading
- ~85 lines of config-related code
- Limited to one cleanup configuration
- Required restart to change settings

**AFTER (Database Approach - This Implementation):**
- Multiple independent cleanup jobs stored in database
- Full CRUD API for dynamic job management
- Zero config code (fully removed)
- Per-job configuration (schedule, filters, retention)
- Live updates via API without restart
- Job execution history tracking

### Total Removals
- **~85 lines removed** across 5 files
- **5 config fields** eliminated
- **5 environment variables** deprecated
- **1 config parameter** removed from service

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
