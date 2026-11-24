package health

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/getmiroir/trading-service/internal/platform"
	_ "modernc.org/sqlite"
)

// ErrNoHeartbeat is returned by ReadLatest when no heartbeat row exists yet.
var ErrNoHeartbeat = errors.New("no service heartbeat recorded")

// Record is a single service heartbeat: the running service version and the
// time it was last written.
type Record struct {
	Version string
	Time    time.Time
}

// DB is a handle to the service health SQLite database. It is used by the
// service to write heartbeats and by the GUI to read them. SQLite WAL mode
// makes the single-writer / single-reader cross-process access safe.
type DB struct {
	sql *sql.DB
}

// Open opens (creating if needed) the health database inside the application
// data directory and ensures the schema exists. WAL mode and a busy timeout
// are enabled so the service and GUI can access the file concurrently.
func Open() (*DB, error) {
	path, err := platform.HealthDBPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening health db: %w", err)
	}

	db := &DB{sql: sqlDB}
	if err := db.ensureSchema(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) ensureSchema() error {
	_, err := db.sql.Exec(`CREATE TABLE IF NOT EXISTS service_health (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		version TEXT NOT NULL,
		time INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("creating service_health table: %w", err)
	}
	return nil
}

// Close closes the underlying database handle.
func (db *DB) Close() error {
	return db.sql.Close()
}

// WriteHeartbeat upserts the single health row with the given version and the
// current time.
func (db *DB) WriteHeartbeat(version string) error {
	return db.writeRow(version, time.Now().UnixMilli())
}

// WriteInstalledVersion records the binary version when the OS service is
// installed. The zero timestamp is not a liveness signal; isAlive treats it as
// stale until the running service overwrites the row via WriteHeartbeat.
func (db *DB) WriteInstalledVersion(version string) error {
	return db.writeRow(version, 0)
}

func (db *DB) writeRow(version string, unixMs int64) error {
	_, err := db.sql.Exec(
		`INSERT INTO service_health (id, version, time) VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET version = excluded.version, time = excluded.time`,
		version, unixMs,
	)
	if err != nil {
		return fmt.Errorf("writing service health row: %w", err)
	}
	return nil
}

// ReadLatest returns the most recent heartbeat. It returns ErrNoHeartbeat when
// the service has never written a row.
func (db *DB) ReadLatest() (Record, error) {
	var version string
	var unixMs int64
	err := db.sql.QueryRow(`SELECT version, time FROM service_health WHERE id = 1`).Scan(&version, &unixMs)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNoHeartbeat
	}
	if err != nil {
		return Record{}, fmt.Errorf("reading heartbeat: %w", err)
	}
	return Record{Version: version, Time: time.UnixMilli(unixMs)}, nil
}

// ClearHeartbeat removes the stored heartbeat row so a subsequent install does
// not read a version left over from a previous service installation.
func (db *DB) ClearHeartbeat() error {
	_, err := db.sql.Exec(`DELETE FROM service_health WHERE id = 1`)
	if err != nil {
		return fmt.Errorf("clearing heartbeat: %w", err)
	}
	return nil
}
