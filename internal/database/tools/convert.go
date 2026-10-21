// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/migrator"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	_ "modernc.org/sqlite"
)

const (
	engineSQLite   = "sqlite"
	enginePostgres = "postgres"
)

// defaultTables is ordered so referenced tables are copied before the tables that reference them.
// sessions and schema_migrations are left out on purpose: sessions only log users out, and the
// target gets its own migration history.
var defaultTables = []string{
	"users",
	"proxy",
	"indexer",
	"indexer_deprecation",
	"irc_network",
	"irc_channel",
	"release_profile_duplicate",
	"filter",
	"filter_external",
	"filter_indexer",
	"client",
	"action",
	"release",
	"release_action_status",
	"release_cleanup_job",
	"notification",
	"filter_notification",
	"notification_inbox",
	"feed",
	"feed_cache",
	"api_key",
	"list",
	"list_filter",
}

// generatedRows are rows that migrations or the first start of autobrr put into an otherwise empty
// database. They are replaced by the source copies, keyed by table with the condition matching them.
var generatedRows = map[string]string{
	"notification":              "type = 'BUILTIN'",
	"release_profile_duplicate": "id IN (1, 2, 3)",
	"indexer_deprecation":       "1 = 1",
}

// sqliteFixups null out references to deleted rows before copying out of SQLite, which does not
// enforce foreign keys. Only needed for FKs with ON DELETE SET NULL.
var sqliteFixups = []string{
	"UPDATE release SET filter_id = NULL WHERE filter_id NOT IN (SELECT id FROM filter)",
	"UPDATE release_action_status SET filter_id = NULL WHERE filter_id NOT IN (SELECT id FROM filter)",
	"UPDATE release_action_status SET action_id = NULL WHERE action_id NOT IN (SELECT id FROM action)",
	"UPDATE indexer SET proxy_id = NULL WHERE proxy_id NOT IN (SELECT id FROM proxy)",
	"UPDATE irc_network SET proxy_id = NULL WHERE proxy_id NOT IN (SELECT id FROM proxy)",
	"UPDATE filter SET release_profile_duplicate_id = NULL WHERE release_profile_duplicate_id NOT IN (SELECT id FROM release_profile_duplicate)",
	"UPDATE action SET client_id = NULL WHERE client_id NOT IN (SELECT id FROM client)",
	"UPDATE feed SET indexer_id = NULL WHERE indexer_id NOT IN (SELECT id FROM indexer)",
	"UPDATE list SET client_id = NULL WHERE client_id NOT IN (SELECT id FROM client)",
}

// Opts controls a conversion run.
type Opts struct {
	ExcludeTables []string
	DryRun        bool
}

// Converter copies all autobrr data between a SQLite and a PostgreSQL database.
type Converter struct {
	log           zerolog.Logger
	sqliteDBPath  string
	postgresDBURL string
}

// NewConverter returns a Converter for the given SQLite file and PostgreSQL DSN.
func NewConverter(log zerolog.Logger, sqliteDBPath, postgresDBURL string) *Converter {
	return &Converter{
		log:           log,
		sqliteDBPath:  sqliteDBPath,
		postgresDBURL: postgresDBURL,
	}
}

type conn struct {
	db     *sql.DB
	engine string
}

func (c conn) placeholder(n int) string {
	if c.engine == enginePostgres {
		return fmt.Sprintf("$%d", n)
	}

	return "?"
}

// maxParams is the bound parameter limit of a single statement.
func (c conn) maxParams() int {
	if c.engine == enginePostgres {
		return 65535
	}

	return 32766
}

func (c conn) migrator(log zerolog.Logger) *migrator.Migrator {
	if c.engine == enginePostgres {
		return migrations.PostgresMigrations(c.db, log)
	}

	return migrations.SQLiteMigrations(c.db, log)
}

// Convert copies a SQLite database into PostgreSQL.
func (c *Converter) Convert(ctx context.Context, opts Opts) error {
	sqliteDB, err := c.openSQLite(true)
	if err != nil {
		return err
	}
	defer sqliteDB.Close()

	postgresDB, err := c.openPostgres(ctx)
	if err != nil {
		return err
	}
	defer postgresDB.Close()

	return c.convert(ctx, conn{db: sqliteDB, engine: engineSQLite}, conn{db: postgresDB, engine: enginePostgres}, opts)
}

// ConvertToSQLite copies a PostgreSQL database into SQLite.
func (c *Converter) ConvertToSQLite(ctx context.Context, opts Opts) error {
	postgresDB, err := c.openPostgres(ctx)
	if err != nil {
		return err
	}
	defer postgresDB.Close()

	if opts.DryRun {
		return c.convert(ctx, conn{db: postgresDB, engine: enginePostgres}, conn{engine: engineSQLite}, opts)
	}

	sqliteDB, err := c.openSQLite(false)
	if err != nil {
		return err
	}
	defer sqliteDB.Close()

	return c.convert(ctx, conn{db: postgresDB, engine: enginePostgres}, conn{db: sqliteDB, engine: engineSQLite}, opts)
}

func (c *Converter) openSQLite(mustExist bool) (*sql.DB, error) {
	if mustExist {
		if _, err := os.Stat(c.sqliteDBPath); err != nil {
			return nil, errors.Wrap(err, "could not find sqlite database: %s", c.sqliteDBPath)
		}
	}

	db, err := sql.Open("sqlite", c.sqliteDBPath+"?_pragma=busy_timeout(10000)")
	if err != nil {
		return nil, errors.Wrap(err, "could not open sqlite database")
	}

	db.SetMaxOpenConns(1)

	return db, nil
}

func (c *Converter) openPostgres(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("postgres", c.postgresDBURL)
	if err != nil {
		return nil, errors.Wrap(err, "could not open postgres database")
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Wrap(err, "could not ping postgres database")
	}

	return db, nil
}

func (c *Converter) convert(ctx context.Context, src, dst conn, opts Opts) error {
	startTime := time.Now()

	l := c.log.With().Str("from", src.engine).Str("to", dst.engine).Logger()

	if err := c.checkSourceSchema(src); err != nil {
		return err
	}

	tables := slices.DeleteFunc(GetTables(), func(table string) bool {
		return slices.Contains(opts.ExcludeTables, table)
	})

	if opts.DryRun {
		for _, table := range tables {
			var count int64
			if err := src.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %q", table)).Scan(&count); err != nil {
				return errors.Wrap(err, "could not count rows in table: %s", table)
			}

			l.Info().Str("table", table).Int64("rows", count).Msg("dry run, would convert table")
		}

		return nil
	}

	if src.engine == engineSQLite {
		c.applyFixups(ctx, src.db, sqliteFixups)
	}

	if err := dst.migrator(c.log).Migrate(); err != nil {
		return errors.Wrap(err, "could not migrate %s database", dst.engine)
	}

	if err := c.prepareTarget(ctx, dst, tables); err != nil {
		return err
	}

	var allFKViolations []string
	for _, table := range tables {
		fkViolations, err := c.convertTable(ctx, l, src, dst, table)
		if err != nil {
			return errors.Wrap(err, "could not convert table: %s", table)
		}

		allFKViolations = append(allFKViolations, fkViolations...)
	}

	if dst.engine == enginePostgres {
		if err := c.resetSequences(ctx, dst, tables); err != nil {
			return err
		}
	}

	printConversionResult(startTime, allFKViolations)

	return nil
}

// checkSourceSchema refuses sources that are not on the latest migration, since their columns
// would not line up with the freshly migrated target.
func (c *Converter) checkSourceSchema(src conn) error {
	m := src.migrator(c.log)

	applied, err := m.CountApplied()
	if err != nil {
		return errors.Wrap(err, "could not read %s schema version, is this an autobrr database?", src.engine)
	}

	if applied != m.TotalMigrations() {
		return errors.New("%s database is at migration %d but %d is expected: start this version of autobrr against it once to migrate it, then convert", src.engine, applied, m.TotalMigrations())
	}

	return nil
}

// prepareTarget makes sure the target holds no user data and drops the generated rows the
// source copies replace.
func (c *Converter) prepareTarget(ctx context.Context, dst conn, tables []string) error {
	for _, table := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %q", table)
		if cond, ok := generatedRows[table]; ok {
			query += " WHERE NOT (" + cond + ")"
		}

		var count int64
		if err := dst.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return errors.Wrap(err, "could not count rows in target table: %s", table)
		}

		if count > 0 {
			return errors.New("target %s database is not empty: table %s has %d rows", dst.engine, table, count)
		}
	}

	for _, table := range tables {
		cond, ok := generatedRows[table]
		if !ok {
			continue
		}

		if _, err := dst.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %q WHERE %s", table, cond)); err != nil {
			return errors.Wrap(err, "could not clear generated rows in target table: %s", table)
		}
	}

	return nil
}

func (c *Converter) convertTable(ctx context.Context, log zerolog.Logger, src, dst conn, table string) ([]string, error) {
	l := log.With().Str("table", table).Logger()

	columns, err := commonColumns(ctx, l, src, dst, table)
	if err != nil {
		return nil, err
	}

	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = fmt.Sprintf("%q", col)
	}
	colNames := strings.Join(quoted, ", ")

	rows, err := src.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %q", colNames, table))
	if err != nil {
		return nil, errors.Wrap(err, "could not query source table")
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, errors.Wrap(err, "could not get column types")
	}

	batchSize := min(1000, dst.maxParams()/len(columns))

	var (
		batch        [][]any
		rowsAffected int64
		fkViolations []string
	)

	flush := func() error {
		inserted, violations, err := c.insertBatch(ctx, l, dst, table, colNames, len(columns), batch)
		if err != nil {
			return err
		}

		rowsAffected += inserted
		fkViolations = append(fkViolations, violations...)
		batch = batch[:0]

		l.Trace().Int64("rows_affected", rowsAffected).Msg("inserted batch")

		return nil
	}

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.Wrap(err, "could not scan source row")
		}

		if dst.engine == engineSQLite {
			for i, v := range values {
				values[i] = toSQLiteValue(v, colTypes[i].DatabaseTypeName())
			}
		}

		batch = append(batch, values)

		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return nil, err
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "could not read source rows")
	}

	if len(batch) > 0 {
		if err := flush(); err != nil {
			return nil, err
		}
	}

	l.Info().Int64("rows_affected", rowsAffected).Msg("converted table")

	return fkViolations, nil
}

// commonColumns returns the source columns that also exist in the target table, so leftovers of
// historic migrations on one side do not break the copy.
func commonColumns(ctx context.Context, l zerolog.Logger, src, dst conn, table string) ([]string, error) {
	srcColumns, err := tableColumns(ctx, src, table)
	if err != nil {
		return nil, err
	}

	dstColumns, err := tableColumns(ctx, dst, table)
	if err != nil {
		return nil, err
	}

	var columns []string
	for _, col := range srcColumns {
		if !slices.Contains(dstColumns, col) {
			l.Debug().Str("column", col).Msg("column missing in target, skipping")
			continue
		}

		columns = append(columns, col)
	}

	if len(columns) == 0 {
		return nil, errors.New("no common columns")
	}

	return columns, nil
}

func tableColumns(ctx context.Context, c conn, table string) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %q WHERE 1 = 0", table))
	if err != nil {
		return nil, errors.Wrap(err, "could not query %s columns", c.engine)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "could not get %s columns", c.engine)
	}

	return columns, nil
}

// toSQLiteValue converts a lib/pq value into the representation the SQLite repos write:
// arrays and json come back as []byte and must be stored as text, not blobs, and timestamps
// are stored as RFC3339 strings.
func toSQLiteValue(v any, pgType string) any {
	switch val := v.(type) {
	case []byte:
		if pgType == "BYTEA" {
			return val
		}

		return string(val)
	case time.Time:
		return val.UTC().Format(time.RFC3339)
	}

	return v
}

func (c *Converter) insertBatch(ctx context.Context, l zerolog.Logger, dst conn, table, colNames string, numColumns int, batch [][]any) (int64, []string, error) {
	placeholders := make([]string, 0, len(batch))
	values := make([]any, 0, len(batch)*numColumns)

	param := 1
	for _, row := range batch {
		rowPlaceholders := make([]string, numColumns)
		for i := range rowPlaceholders {
			rowPlaceholders[i] = dst.placeholder(param)
			param++
		}

		placeholders = append(placeholders, "("+strings.Join(rowPlaceholders, ", ")+")")
		values = append(values, row...)
	}

	query := fmt.Sprintf("INSERT INTO %q (%s) VALUES %s", table, colNames, strings.Join(placeholders, ", "))

	if _, err := dst.db.ExecContext(ctx, query, values...); err != nil {
		if isForeignKeyViolation(err) {
			l.Debug().Msg("batch violates foreign key, inserting rows one by one")
			return c.insertOneByOne(ctx, dst, table, colNames, numColumns, batch)
		}

		return 0, nil, errors.Wrap(err, "could not insert batch")
	}

	return int64(len(batch)), nil, nil
}

// insertOneByOne isolates the rows of a batch that violate foreign keys so the rest still get copied.
func (c *Converter) insertOneByOne(ctx context.Context, dst conn, table, colNames string, numColumns int, batch [][]any) (int64, []string, error) {
	placeholders := make([]string, numColumns)
	for i := range placeholders {
		placeholders[i] = dst.placeholder(i + 1)
	}

	stmt, err := dst.db.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)", table, colNames, strings.Join(placeholders, ", ")))
	if err != nil {
		return 0, nil, errors.Wrap(err, "could not prepare insert")
	}
	defer stmt.Close()

	var (
		rowsAffected int64
		fkViolations []string
	)

	for _, row := range batch {
		if _, err := stmt.ExecContext(ctx, row...); err != nil {
			if isForeignKeyViolation(err) {
				fkViolations = append(fkViolations, fmt.Sprintf("Table '%s': %v", table, err))
				continue
			}

			return rowsAffected, fkViolations, errors.Wrap(err, "could not insert row")
		}

		rowsAffected++
	}

	return rowsAffected, fkViolations, nil
}

// resetSequences moves every serial sequence past the copied ids, otherwise the next insert
// collides with an existing row.
func (c *Converter) resetSequences(ctx context.Context, dst conn, tables []string) error {
	for _, table := range tables {
		columns, err := tableColumns(ctx, dst, table)
		if err != nil {
			return err
		}

		if !slices.Contains(columns, "id") {
			continue
		}

		query := fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM %q`, table, table)
		if _, err := dst.db.ExecContext(ctx, query); err != nil {
			return errors.Wrap(err, "could not reset sequence for table: %s", table)
		}
	}

	return nil
}

func (c *Converter) applyFixups(ctx context.Context, db *sql.DB, stmts []string) {
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			c.log.Error().Err(err).Str("statement", stmt).Msg("could not apply fixup")
		}
	}
}

func printConversionResult(startTime time.Time, allFKViolations []string) {
	var sb strings.Builder

	sb.WriteString("Convert completed successfully!\n")
	fmt.Fprintf(&sb, "Elapsed time: %s\n", time.Since(startTime))
	if len(allFKViolations) > 0 {
		sb.WriteString("\nSummary of Foreign Key Violations:\n\n")
		for _, msg := range allFKViolations {
			fmt.Fprintf(&sb, " - %s\n", msg)
		}
		sb.WriteString("\nThese are due to missing references, likely because the related item in another table no longer exists.\n")
	}

	fmt.Print(sb.String())
}

// GetTables returns the tables copied by a conversion, in insert order.
func GetTables() []string {
	return slices.Clone(defaultTables)
}

func isForeignKeyViolation(err error) bool {
	return strings.Contains(err.Error(), "violates foreign key constraint")
}
