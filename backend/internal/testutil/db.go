package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewTestDB creates a new Postgres test container, runs migrations, and returns a connection pool.
// The container is automatically cleaned up when the test finishes.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	// Start Postgres container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("vmail_test"),
		postgres.WithUsername("vmail"),
		postgres.WithPassword("vmail"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start Postgres container: %v", err)
	}

	// Cleanup function
	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Errorf("Failed to terminate container: %v", err)
		}
	})

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Create connection pool with same configuration as production
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		t.Fatalf("Failed to parse connection string: %v", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return pool
}

// runMigrations reads all migration files and executes them in order
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Read all migration files
	migrations, err := readMigrations()
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// Execute migrations in order
	for _, migration := range migrations {
		if _, err := pool.Exec(ctx, migration.SQL); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

type migration struct {
	Name string
	SQL  string
}

// readMigrations reads all .up.sql files from the migrations directory and returns them sorted by filename
func readMigrations() ([]migration, error) {
	var migrations []migration

	// Get the "migrations" directory path relative to the backend root
	// Try multiple possible locations based on where the test is run from
	possiblePaths := []string{
		"migrations",          // From the backend directory
		"backend/migrations",  // From the project root
		"../migrations",       // From internal/testutil
		"../../migrations",    // From internal/testutil
		"../../../migrations", // From deeper packages
	}

	var migrationsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			migrationsDir = path
			break
		}
	}

	if migrationsDir == "" {
		return nil, fmt.Errorf("migrations directory not found. Tried: %v", possiblePaths)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			path := filepath.Join(migrationsDir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %w", path, err)
			}

			migrations = append(migrations, migration{
				Name: entry.Name(),
				SQL:  string(content),
			})
		}
	}

	// Sort migrations by filename to ensure the correct order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}
