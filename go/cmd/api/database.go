package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// How long Open waits for the database to become reachable, and the ceiling on the
// backoff between attempts.
//
// # Why waiting at all
//
// A single ping and then `PrintFatal` makes the API's boot depend on winning a race it has
// no control over. Every `docker compose up` starts the API and MySQL together, and MySQL
// is routinely the slower of the two — so the API dies with `dial tcp …:3306: connect:
// connection refused` before the database has finished starting. In production a restart
// policy papers over it; in dev the hot-reload loop only rebuilds on a **file change**, so
// nothing ever restarts the process and the API stays dead until somebody notices. The UI
// still serves its shell, so what the operator sees is a page that loads and then fails
// every request — which looks like a broken feature rather than a missing database.
//
// # Why bounded, and why it still fails
//
// Retrying forever would turn a wrong password into a process that hangs looking healthy.
// The budget is generous enough to cover a cold MySQL start and short enough that a real
// misconfiguration is a failure rather than a mystery — and every attempt is logged, so a
// permanent error like access-denied is visible immediately instead of at the end.
const (
	defaultConnectBudget  = 90 * time.Second
	defaultConnectMaxWait = 5 * time.Second
)

type DatabaseConfig struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type database struct {
	db  *sql.DB
	cfg DatabaseConfig

	// Fields rather than constants so a test can shorten them; nothing else sets them.
	connectBudget  time.Duration
	connectMaxWait time.Duration
}

func NewDatabase(cfg DatabaseConfig) *database {
	return &database{
		cfg:            cfg,
		connectBudget:  defaultConnectBudget,
		connectMaxWait: defaultConnectMaxWait,
	}
}

func (db *database) DB() *sql.DB {
	return db.db
}

func (db *database) Migrate() error {
	migrationDriver, err := postgres.WithInstance(db.db, &postgres.Config{})
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithDatabaseInstance("file:///app/migrations", "postgres", migrationDriver)
	if err != nil {
		return err
	}
	err = migrator.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (db *database) Dialect() string {
	return "mysql"
}

func (db *database) Open() (err error) {
	db.db, err = sql.Open(db.Dialect(), db.cfg.dsn)
	if err != nil {
		return err
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.db.SetMaxOpenConns(db.cfg.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.db.SetMaxIdleConns(db.cfg.maxIdleConns)

	duration, err := time.ParseDuration(db.cfg.maxIdleTime)
	if err != nil {
		return err
	}
	db.db.SetConnMaxIdleTime(duration)

	// sql.Open never contacts the server, so this is where a missing database actually
	// shows up.
	return db.waitForConnection()
}

// waitForConnection pings until the database answers, the budget runs out, or the error is
// clearly not going to fix itself.
//
// Every failure is retried rather than only the ones that look transient. "Connection
// refused" is the obvious case, but a first boot can also legitimately report an unknown
// database while the server's init scripts are still creating it — and classifying driver
// errors by hand is how a retry loop quietly stops covering the case it was written for.
// The budget is what makes retrying everything safe.
func (db *database) waitForConnection() error {
	deadline := time.Now().Add(db.connectBudget)
	// Capped from the start: the first wait obeys the ceiling like every other one, so the
	// backoff cannot begin above its own maximum.
	wait := min(time.Second, db.connectMaxWait)

	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := db.db.PingContext(ctx)
		cancel()

		if err == nil {
			if attempt > 1 {
				log.Printf("database: reachable after %d attempts", attempt)
			}
			return nil
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("database unreachable after %d attempts over %s: %w", attempt, db.connectBudget, err)
		}

		// Never sleep past the deadline: the budget is a promise about total time, and
		// overshooting it by most of a backoff step would break that for no benefit.
		if wait > remaining {
			wait = remaining
		}
		log.Printf("database: not ready (attempt %d): %v — retrying in %s", attempt, err, wait)
		time.Sleep(wait)
		wait = min(wait*2, db.connectMaxWait)
	}
}

func (db *database) Close() error {
	return db.db.Close()
}

func (db *database) Stats() sql.DBStats {
	return db.db.Stats()
}
