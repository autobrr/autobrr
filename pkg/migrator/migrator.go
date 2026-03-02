package migrator

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

type SQLDB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Begin() (*sql.Tx, error)
	Close() error
	Driver() driver.Driver
}

const DefaultTableName = "schema_migrations"

const (
	EngineSQLite   = "sqlite"
	EnginePostgres = "postgres"
)

type Migrator struct {
	db             *sql.DB
	logger         Logger
	embedFS        *embed.FS
	squirrel       sq.StatementBuilderType
	engine         string
	tableName      string
	filepathPrefix string

	initialSchemaFile string
	initialSchema     string

	migrations          []*Migration
	migrationLookup     map[int]*Migration
	migrationNameLookup map[string]*Migration

	PreMigrationHook func() error
}

type Option func(migrate *Migrator)

func WithEngine(engine string) Option {
	return func(migrate *Migrator) {
		migrate.engine = engine
	}
}

func WithTableName(table string) Option {
	return func(migrate *Migrator) {
		migrate.tableName = table
	}
}

func WithSchemaString(schema string) Option {
	return func(migrate *Migrator) {
		migrate.initialSchema = schema
	}
}

func WithSchemaFile(file string) Option {
	return func(migrate *Migrator) {
		migrate.initialSchemaFile = file
	}
}

func WithEmbedFS(embedFS embed.FS, prefix string) Option {
	return func(migrate *Migrator) {
		migrate.embedFS = &embedFS
		//dir, _ := migrate.embedFS.ReadDir(".")
		migrate.filepathPrefix = prefix
	}
}

func WithPreMigrationHook(hook func() error) Option {
	return func(migrate *Migrator) {
		migrate.PreMigrationHook = hook
	}
}

// Logger interface
type Logger interface {
	Printf(string, ...interface{})
}

// LoggerFunc adapts Logger and any third party logger
type LoggerFunc func(string, ...interface{})

// Printf implements Logger interface
func (f LoggerFunc) Printf(msg string, args ...interface{}) {
	f(msg, args...)
}

func WithLogger(logger Logger) Option {
	return func(migrate *Migrator) {
		migrate.logger = logger
	}
}

func NewMigrate(db *sql.DB, opts ...Option) *Migrator {
	m := &Migrator{
		db:                  db,
		tableName:           DefaultTableName,
		logger:              log.New(io.Discard, "migrator: ", 0),
		squirrel:            sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		initialSchema:       "",
		initialSchemaFile:   "",
		migrations:          make([]*Migration, 0),
		migrationLookup:     map[int]*Migration{},
		migrationNameLookup: map[string]*Migration{},
		PreMigrationHook:    nil,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

type Migration struct {
	id int

	Name  string
	File  string
	Run   func(db *sql.DB) error
	RunTx func(db *sql.Tx) error

	db *sql.DB
}

func (m *Migration) String() string {
	return m.Name
}

func (m *Migration) Id() int {
	return m.id
}

func (m *Migrator) TableDrop(table string) error {
	if _, err := m.db.Exec(fmt.Sprintf(`DROP TABLE "%s"`, table)); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) Add(mi ...*Migration) {
	for _, migration := range mi {
		//id := len(m.migrations) + 1
		id := 0
		if len(m.migrations) > 0 {
			id = len(m.migrations) + 1
		}
		migration.db = m.db
		migration.id = id

		m.migrations = append(m.migrations, migration)
		//m.migrationLookup[len(m.migrations)+1] = migration
		m.migrationLookup[id] = migration
		m.migrationNameLookup[migration.Name] = migration
	}
}

func (m *Migrator) AddMigration(mi *Migration) {
	id := 0
	if len(m.migrations) > 0 {
		id = len(m.migrations) + 1
	}
	//id := len(m.migrations) + 1
	mi.db = m.db
	mi.id = id

	m.migrations = append(m.migrations, mi)
	//m.migrationLookup[len(m.migrations)+1] = mi
	m.migrationLookup[id] = mi
	m.migrationNameLookup[mi.Name] = mi
}

func (m *Migrator) AddFileMigration(file string) {
	//id := len(m.migrations) + 1
	id := 0
	if len(m.migrations) > 0 {
		id = len(m.migrations) + 1
	}
	name := strings.TrimSuffix(file, ".sql")

	mi := &Migration{
		id:   id,
		db:   m.db,
		Name: name,
		File: file,
	}

	m.migrations = append(m.migrations, mi)
	//m.migrationLookup[len(m.migrations)+1] = mi
	m.migrationLookup[id] = mi
	m.migrationNameLookup[mi.Name] = mi
}

func (m *Migrator) Get(name string) (*Migration, error) {
	migration, ok := m.migrationNameLookup[name]
	if !ok {
		return nil, errors.New("migration not found")
	}
	return migration, nil
}

func (m *Migrator) GetById(id int) (*Migration, error) {
	migration, ok := m.migrationLookup[id]
	if !ok {
		return nil, errors.New("migration not found")
	}
	return migration, nil
}

func (m *Migrator) GetUpTo(name string) []*Migration {
	var migrations []*Migration
	for _, migration := range m.migrations {
		migrations = append(migrations, migration)

		if migration.Name == name {
			return migrations
		}
	}

	return migrations
}

func (m *Migrator) GetUpToId(id int) []*Migration {
	var migrations []*Migration
	for _, migration := range m.migrations {
		if migration.id > id {
			break
		}

		migrations = append(migrations, migration)
	}
	return migrations
}

func (m *Migrator) Exec(query string, args ...string) error {
	if _, err := m.db.Exec(query, args); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) BeginTx() (*sql.Tx, error) {
	return m.db.BeginTx(context.Background(), nil)
}

func (m *Migrator) TotalMigrations() int {
	return len(m.migrations)
}

// convertMigrationsTableSingleToMultiPG converts a single-row version table to a multi-row table
func (m *Migrator) convertMigrationsTableSingleToMultiPG() error {
	if err := m.initVersionTable(); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	var count int
	query, args, err := m.squirrel.Select("COUNT(*)").From(m.tableName).ToSql()
	if err != nil {
		return err
	}
	err = m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	if count > 1 {
		// all is well, so lets return
		return nil
	}

	var appliedMigration struct {
		id      int
		version int
	}
	query, args, err = m.squirrel.Select("id", "version").From(m.tableName).Limit(1).ToSql()
	if err != nil {
		return err
	}

	if err = m.db.QueryRow(query, args...).Scan(&appliedMigration.id, &appliedMigration.version); err != nil {
		return err
	}

	if err := m.migrateOldVersionTable(appliedMigration.version); err != nil {
		return err
	}

	return nil
}

// convertMigrationsTableSingleToMultiSQLite converts sqlite PRAGMA to a multi-row table
func (m *Migrator) convertMigrationsTableSingleToMultiSQLite() error {
	if err := m.initVersionTable(); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	var count int
	query, args, err := m.squirrel.Select("COUNT(*)").From(m.tableName).ToSql()
	if err != nil {
		return err
	}

	err = m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return err
	}

	var version int
	if err := m.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}

	// If we have no migrations, and no version, then we're done
	if count == 0 && version == 0 {
		return nil
	}

	if count == 0 && version > 0 {
		// We have an old pragma-based version, need to migrate
		return m.migratePragmaToNewTable(version)
	}

	return nil
}

func (m *Migrator) checkPreReqs() error {
	switch m.engine {
	case EnginePostgres:
		if err := m.convertMigrationsTableSingleToMultiPG(); err != nil {
			return err
		}
		break
	case EngineSQLite:
		if err := m.convertMigrationsTableSingleToMultiSQLite(); err != nil {
			return err
		}
		break
	}

	return nil
}

// migratePragmaToNewTable migrates from SQLite PRAGMA user_version to new multi-row table
func (m *Migrator) migratePragmaToNewTable(currentVersion int) error {
	m.logger.Printf("migrating from PRAGMA user_version (%d) to multi-row schema_migrations table", currentVersion)

	if m.PreMigrationHook != nil {
		m.logger.Printf("running pre migration hook to backup database...")

		if err := m.PreMigrationHook(); err != nil {
			return errors.Wrap(err, "migrator: could not run pre migration hook")
		}
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	// Create new table
	if err := m.initVersionTableTx(tx); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	migrations := m.GetUpToId(currentVersion)
	if len(migrations) == 0 {
		return nil
	}

	m.logger.Printf("found %d migrations to convert to new table format", len(migrations))

	if err := m.updateSchemaVersions(tx, migrations); err != nil {
		return err
	}

	m.logger.Printf("successfully migrated %d migrations to new table format", currentVersion)

	// Reset pragma to 0 since we're no longer using it
	if _, err = tx.Exec("PRAGMA user_version = 0"); err != nil {
		return errors.Wrap(err, "error resetting PRAGMA user_version")
	}

	m.logger.Printf("successfully migrated %d migrations from PRAGMA to table", currentVersion)

	return err
}

// migrateOldVersionTable migrates from old single-row version table to new multi-row table
func (m *Migrator) migrateOldVersionTable(currentVersion int) error {
	m.logger.Printf("migrating from old single-row version table to multi-row format")

	m.logger.Printf("current version in old format: %d", currentVersion)

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	// Drop old table
	if _, err = tx.Exec(fmt.Sprintf("DROP TABLE %s", m.tableName)); err != nil {
		return errors.Wrapf(err, "could not drop old version table: %s", m.tableName)
	}

	if err := m.initVersionTableTx(tx); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	migrations := m.GetUpToId(currentVersion)
	if len(migrations) == 0 {
		return nil
	}

	m.logger.Printf("found %d migrations to convert to new table format", len(migrations))

	if err := m.updateSchemaVersions(tx, migrations); err != nil {
		return err
	}

	m.logger.Printf("successfully migrated %d migrations to new table format", currentVersion)

	return err
}

func (m *Migrator) updateSchemaVersions(tx *sql.Tx, migrations []*Migration) error {
	for _, migration := range migrations {
		if err := m.updateSchemaVersionTx(tx, migration.id, migration.Name); err != nil {
			return err
		}
	}

	return nil
}

// CountApplied Count the number of rows in the migrations table
func (m *Migrator) CountApplied() (int, error) {
	var count int

	err := m.squirrel.Select("COUNT(*)").From(m.tableName).RunWith(m.db).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (m *Migrator) GetPendingMigrations() ([]*Migration, error) {
	count, err := m.CountApplied()
	if err != nil {
		return nil, err
	}

	return m.migrations[count:len(m.migrations)], nil
}

func (m *Migrator) initVersionTable() error {
	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id      INTEGER NOT NULL PRIMARY KEY,
			version VARCHAR(255) NOT NULL
		)`, m.tableName)

	_, err := m.db.Exec(createTable)
	if err != nil {
		return errors.Wrapf(err, "migrator: could not create version table: %s", m.tableName)
	}

	return nil
}

func (m *Migrator) initVersionTableTx(tx *sql.Tx) error {
	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id      INTEGER NOT NULL PRIMARY KEY,
			version VARCHAR(255) NOT NULL
		)`, m.tableName)

	_, err := tx.Exec(createTable)
	if err != nil {
		return errors.Wrapf(err, "migrator: could not create version table: %s", m.tableName)
	}

	return nil
}

func (m *Migrator) InitVersionTable() error {
	return m.initVersionTable()
}

func (m *Migrator) RunMigrations(migrations []*Migration) error {
	for _, migration := range migrations {
		if err := m.migrate(migration.id, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) RunMigration(migration *Migration) error {
	if err := m.migrate(migration.id, migration); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) Migrate() error {
	if err := m.checkPreReqs(); err != nil {
		return errors.Wrap(err, "migrator: could not check pre-reqs")
	}

	appliedCount, err := m.CountApplied()
	if err != nil {
		return errors.Wrap(err, "migrator: could not get applied migrations count")
	}

	if appliedCount == 0 && (m.initialSchema != "" || m.initialSchemaFile != "") {
		m.logger.Printf("preparing to apply base schema migration")

		if err := m.migrateInitialSchema(); err != nil {
			return errors.Wrap(err, "migrator: could not apply base schema")
		}

		return nil
	}

	if appliedCount > len(m.migrations) {
		return errors.New("migrator: applied migration number on db cannot be greater than the defined migration list")
	}

	if appliedCount == len(m.migrations) {
		m.logger.Printf("database schema up to date")
		return nil
	}

	if m.PreMigrationHook != nil {
		if err := m.PreMigrationHook(); err != nil {
			return errors.Wrap(err, "migrator: could not run pre migration hook")
		}
	}

	migrationsToApply, err := m.GetPendingMigrations()
	if err != nil {
		return errors.Wrap(err, "migrator: could not get pending migrations")
	}

	if len(migrationsToApply) == 0 {
		m.logger.Printf("database schema up to date")
		return nil
	}

	m.logger.Printf("found %d migrations to apply", len(migrationsToApply))

	for _, migration := range migrationsToApply {
		if err := m.migrate(migration.id, migration); err != nil {
			return errors.Wrapf(err, "migrator: error while running migration: %s", migration.String())
		}
	}

	m.logger.Printf("successfully applied all migrations!")

	return nil
}

func (m *Migrator) updateSchemaVersionTx(tx *sql.Tx, id int, version string) error {
	_, err := m.squirrel.Insert(m.tableName).Columns("id", "version").Values(id, version).RunWith(tx).Exec()
	if err != nil {
		return errors.Wrapf(err, "error inserting migration version: %s", version)
	}

	return nil
}

func (m *Migrator) updateSchemaVersion(id int, version string) error {
	_, err := m.squirrel.Insert(m.tableName).Columns("id", "version").Values(id, version).RunWith(m.db).Exec()
	if err != nil {
		return errors.Wrapf(err, "error inserting migration version: %s", version)
	}

	return nil
}

// readFile from embed.FS if provided or local fs as a fallback
func (m *Migrator) readFile(filename string) ([]byte, error) {
	if m.embedFS != nil {
		//d, err := m.embedFS.ReadDir(".")
		//if err != nil {
		//	return nil, errors.Wrapf(err, "could not read initial schema file %q from embed.FS", filename)
		//}
		//m.logger.Printf("found %d files in embed.FS", len(d))
		//
		//if len(d) == 0 {
		//	return nil, errors.New("embed.FS: no files found")
		//}
		migrationFile := filename
		if m.filepathPrefix != "" {
			// use old path package since embed always use forward slash
			migrationFile = path.Join(m.filepathPrefix, migrationFile)
		}

		data, err := m.embedFS.ReadFile(migrationFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read initial schema file (%s) from embed.FS", filename)
		}

		return data, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read initial schema: %s", filename)
	}

	return data, nil
}

func (m *Migrator) migrateInitialSchema() error {
	if m.initialSchema == "" && m.initialSchemaFile != "" {
		data, err := m.readFile(m.initialSchemaFile)
		if err != nil {
			return errors.Wrapf(err, "could not read initial schema: %s", m.initialSchemaFile)
		}

		m.initialSchema = string(data)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	m.logger.Printf("applying base schema migration...")

	if _, err = tx.Exec(m.initialSchema); err != nil {
		return errors.Wrap(err, "error applying base schema migration")
	}

	if err := m.updateSchemaVersions(tx, m.migrations); err != nil {
		return errors.Wrap(err, "error updating migration versions")
	}

	m.logger.Printf("applied base schema migration")

	return err
}

func (m *Migrator) migrate(migrationNumber int, migration *Migration) error {
	if migration.Name == "" {
		return errors.New("migration must have a name")
	}

	if migration.Run == nil && migration.RunTx == nil && migration.File == "" {
		return errors.New("migration must have a Run/RunTx function or a valid File path")
	}

	if migration.Run != nil && migration.File != "" {
		return errors.New("migration cannot have both Run function and File path")
	} else if migration.RunTx != nil && migration.File != "" {
		return errors.New("migration cannot have both RunTx function and File path")
	}

	if migration.Run != nil {
		m.logger.Printf("applying migration: %s from Run", migration.Name)
		if err := migration.Run(m.db); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

		if err := m.updateSchemaVersion(migrationNumber, migration.Name); err != nil {
			return errors.Wrapf(err, "error updating migration versions: %s", migration.Name)
		}

		return nil
	}

	tx, err := m.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error could not begin transaction")
	}

	defer func() {
		if err != nil {
			if errRb := tx.Rollback(); errRb != nil {
				//err = fmt.Errorf("error rolling back: %s\n%s", errRb, err)
				err = errors.Wrapf(errRb, "error rolling back: %q", err)
			}
			return
		}
		err = tx.Commit()
	}()

	//m.logger.Printf("applying migration: %s", migration.Name)

	//if migration.Run != nil {
	//	m.logger.Printf("applying migration: %s from Run", migration.Name)
	//	if err = migration.Run(m.db); err != nil {
	//		return errors.Wrapf(err, "error executing migration: %s", migration.Name)
	//	}
	//
	//} else
	if migration.RunTx != nil {
		m.logger.Printf("applying migration: %s from RunTx", migration.Name)
		if err = migration.RunTx(tx); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.File != "" {
		m.logger.Printf("applying migration: %s from file: %s", migration.Name, migration.File)

		// handle file based migration
		data, err := m.readFile(migration.File)
		if err != nil {
			return errors.Wrapf(err, "could not read migration from file: %s", migration.File)
		}

		if _, err = tx.Exec(string(data)); err != nil {
			return errors.Wrapf(err, "error applying schema migration from file: %s", migration.File)
		}
	}

	if err = m.updateSchemaVersionTx(tx, migrationNumber, migration.Name); err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", migration.Name)
	}

	//m.logger.Printf("applied migration: %s", migration.Name)

	return err
}
