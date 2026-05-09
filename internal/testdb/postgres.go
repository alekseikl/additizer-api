// Package testdb opens a PostgreSQL database for integration-style tests.
//
// Start the test database before running tests, for example:
//
//	docker compose up -d postgres_test
//
// Override the DSN with TEST_DATABASE_URL if needed.
package testdb

import (
	"os"
	"sync"
	"testing"

	"github.com/alekseikl/additizer-api/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultTestDSN = "host=localhost port=5433 user=postgres password=postgres dbname=additizer_test sslmode=disable"

// Lock key for pg_advisory_lock so only one test process (go test runs packages in separate binaries)
// uses the shared database at a time.
const testDBAdvisoryKey int64 = 0xADD17A7040

var migrateOnce sync.Once

// Open connects to the test Postgres, ensures schema is migrated once, and returns a DB
// handle with all tables truncated for a clean slate. The underlying sql.DB is closed in t.Cleanup.
func Open(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf(
			"open test database: %v\n"+
				"Set TEST_DATABASE_URL or start the test container, e.g.: docker compose up -d postgres_test",
			err,
		)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(4)

	if err := db.Exec("SELECT pg_advisory_lock(?)", testDBAdvisoryKey).Error; err != nil {
		_ = sqlDB.Close()
		t.Fatalf("acquire test database lock: %v", err)
	}

	var migrateErr error
	migrateOnce.Do(func() {
		migrateErr = database.Migrate(db)
	})
	if migrateErr != nil {
		_ = db.Exec("SELECT pg_advisory_unlock(?)", testDBAdvisoryKey).Error
		_ = sqlDB.Close()
		t.Fatalf("migrate test database: %v", migrateErr)
	}

	if err := truncateAll(db); err != nil {
		_ = db.Exec("SELECT pg_advisory_unlock(?)", testDBAdvisoryKey).Error
		_ = sqlDB.Close()
		t.Fatalf("truncate test tables: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Exec("SELECT pg_advisory_unlock(?)", testDBAdvisoryKey).Error; err != nil {
			t.Fatalf("release test database lock: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})

	return db
}

func truncateAll(db *gorm.DB) error {
	// Child tables first; RESTART IDENTITY clears serial/identity for stable ids in tests.
	return db.Exec(`
		TRUNCATE TABLE
			preset_shares,
			preset_group_shares,
			presets,
			preset_groups,
			users
		RESTART IDENTITY CASCADE
	`).Error
}
