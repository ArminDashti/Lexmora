package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

// timeLayout matches the strftime('%Y-%m-%d %H:%M:%f') defaults stored in SQLite,
// so text comparisons in WHERE clauses stay correct.
const timeLayout = "2006-01-02 15:04:05.000"

func TimeText(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// ponytail: one connection serializes all queries (SQLite allows a single writer).
	// Ceiling: reads queue behind each other. Upgrade: separate read pool under WAL.
	conn.SetMaxOpenConns(1)
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return conn, nil
}

func RunMigrations(path string) error {
	m, err := migrate.New("file://migrations", "sqlite://"+path)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
