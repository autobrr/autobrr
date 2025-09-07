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
	"path/filepath"
	"strings"

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

type Migrator struct {
	db             *sql.DB
	logger         Logger
	embedFS        *embed.FS
	tableName      string
	filepathPrefix string

	// used for SQLite only
	usePragma bool
	pragmaKey string

	initialSchemaFile string
	initialSchema     string

	migrations          []*Migration
	migrationLookup     map[int]*Migration
	migrationNameLookup map[string]*Migration
}

type Option func(migrate *Migrator)

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

func WithEmbedFS(embedFS embed.FS) Option {
	return func(migrate *Migrator) {
		migrate.embedFS = &embedFS
		//dir, _ := migrate.embedFS.ReadDir(".")
	}
}

func WithSQLitePragma(key string) Option {
	return func(migrate *Migrator) {
		migrate.usePragma = true
		migrate.pragmaKey = key
	}
}

func WithFilePathPrefix(prefix string) Option {
	return func(migrate *Migrator) {
		migrate.filepathPrefix = prefix
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
		initialSchema:       "",
		initialSchemaFile:   "",
		migrations:          make([]*Migration, 0),
		migrationLookup:     map[int]*Migration{},
		migrationNameLookup: map[string]*Migration{},
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
		id := len(m.migrations) + 1
		migration.db = m.db
		migration.id = id

		m.migrations = append(m.migrations, migration)
		//m.migrationLookup[len(m.migrations)+1] = migration
		m.migrationLookup[id] = migration
		m.migrationNameLookup[migration.Name] = migration
	}
}

func (m *Migrator) AddMigration(mi *Migration) {
	id := len(m.migrations) + 1
	mi.db = m.db
	mi.id = id

	m.migrations = append(m.migrations, mi)
	//m.migrationLookup[len(m.migrations)+1] = mi
	m.migrationLookup[id] = mi
	m.migrationNameLookup[mi.Name] = mi
}

func (m *Migrator) AddFileMigration(file string) {
	id := len(m.migrations) + 1
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

func (m *Migrator) Exec(query string, args ...string) error {
	if _, err := m.db.Exec(query, args); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) BeginTx() (*sql.Tx, error) {
	return m.db.BeginTx(context.Background(), nil)
}

func (m *Migrator) CountApplied() (int, error) {
	var count int

	if m.usePragma && m.pragmaKey != "" {
		if err := m.db.QueryRow("PRAGMA user_version").Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}

	rows, err := m.db.Query(fmt.Sprintf("SELECT count(*) FROM %s", m.tableName))
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return count, nil
}

func (m *Migrator) Pending() ([]*Migration, error) {
	count, err := m.CountApplied()
	if err != nil {
		return nil, err
	}

	return m.migrations[count:len(m.migrations)], nil
}

func (m *Migrator) InitVersionTable() error {
	if !m.usePragma && m.tableName != "" {
		migrationsTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
    id      INT8 NOT NULL,
	version VARCHAR(255) NOT NULL,
	PRIMARY KEY(id)
);`, m.tableName)

		_, err := m.db.Exec(migrationsTable)
		if err != nil {
			return errors.Wrapf(err, "migrator: could not create version table: %s", m.tableName)
		}
	}

	return nil
}

func (m *Migrator) RunMigrations(migrations []*Migration) error {
	// TODO should this be done here?
	if err := m.InitVersionTable(); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	for _, migration := range migrations {
		if err := m.migrate(migration.id, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) RunMigration(migration *Migration) error {
	if err := m.InitVersionTable(); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	if err := m.migrate(migration.id, migration); err != nil {
		return err
	}

	return nil
}

func (m *Migrator) Migrate() error {
	if err := m.InitVersionTable(); err != nil {
		return errors.Wrap(err, "migrator: could not init version table")
	}

	appliedCount, err := m.CountApplied()
	if err != nil {
		return errors.Wrap(err, "migrator: could not get applied migrations count")
	}

	if appliedCount == 0 && m.initialSchema != "" {
		m.logger.Printf("preparing to apply base schema migration")

		if err := m.migrateInitialSchema(); err != nil {
			return errors.Wrap(err, "migrator: could not apply base schema")
		}

		return nil
	}

	// TODO check base schema migrations++
	//if appliedCount-1 > len(m.migrations) {
	if appliedCount > len(m.migrations) {
		return errors.New("migrator: applied migration number on db cannot be greater than the defined migration list")
	}

	if appliedCount == len(m.migrations) {
		m.logger.Printf("database schema up to date")
		return nil
	}

	//for idx, migration := range m.migrations[appliedCount-1 : len(m.migrations)] {
	for idx, migration := range m.migrations[appliedCount:len(m.migrations)] {
		if err := m.migrate(idx+appliedCount, migration); err != nil {
			return errors.Wrapf(err, "migrator: error while running migration: %s", migration.String())
		}
	}

	m.logger.Printf("successfully applied all migrations!")

	return nil
}

func (m *Migrator) updateSchemaVersion(tx *sql.Tx, id int, version string) error {
	updateVersion := fmt.Sprintf("INSERT INTO %s (id, version) VALUES (%d, '%s')", m.tableName, id, version)
	if m.usePragma && m.pragmaKey != "" {
		updateVersion = fmt.Sprintf("PRAGMA user_version = %d", id)
	}

	_, err := tx.Exec(updateVersion)
	if err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", version)
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
			migrationFile = filepath.Join(m.filepathPrefix, migrationFile)
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

	if err = m.updateSchemaVersion(tx, 0, "initial schema"); err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", "initial schema")
	}

	//if len(m.migrations) > 0 {
	//	lastMigration := m.migrations[len(m.migrations)-1]
	//
	//	if err = m.updateVersion(tx, len(m.migrations), lastMigration.Name); err != nil {
	//		return errors.Wrapf(err, "error updating migration versions: %s", lastMigration.Name)
	//	}
	//}

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

	if migration.Run != nil {
		m.logger.Printf("applying migration: %s from Run", migration.Name)
		if err = migration.Run(m.db); err != nil {
			return errors.Wrapf(err, "error executing migration: %s", migration.Name)
		}

	} else if migration.RunTx != nil {
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

	if err = m.updateSchemaVersion(tx, migrationNumber, migration.Name); err != nil {
		return errors.Wrapf(err, "error updating migration versions: %s", migration.Name)
	}

	//m.logger.Printf("applied migration: %s", migration.Name)

	return err
}
