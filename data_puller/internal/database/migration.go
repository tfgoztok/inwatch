package database

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// MigrationManager manages database migrations
type MigrationManager struct {
	dsn string // Data Source Name for the database connection
}

// NewMigrationManager creates a new MigrationManager with the provided database connection parameters
func NewMigrationManager(host, port, user, password, dbname string) *MigrationManager {
	// Construct the DSN for connecting to the PostgreSQL database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
	return &MigrationManager{dsn: dsn} // Return a new instance of MigrationManager
}

// Run applies all available migrations to the database
func (m *MigrationManager) Run() error {
	// Create a new migration instance
	migration, err := migrate.New(
		"file://migrations", // Path to migration files
		m.dsn,               // Database connection string
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err) // Handle error
	}
	defer migration.Close() // Ensure the migration instance is closed after use

	// Apply the migrations
	if err := migration.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No migration changes needed") // Log if no changes are needed
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err) // Handle error
	}

	return nil // Return nil if migrations were successful
}

// Rollback reverts the last applied migration
func (m *MigrationManager) Rollback() error {
	// Create a new migration instance
	migration, err := migrate.New(
		"file://migrations", // Path to migration files
		m.dsn,               // Database connection string
	)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err) // Handle error
	}
	defer migration.Close() // Ensure the migration instance is closed after use

	// Revert the last migration
	if err := migration.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No rollback needed") // Log if no rollback is needed
			return nil
		}
		return fmt.Errorf("failed to rollback migrations: %w", err) // Handle error
	}

	return nil // Return nil if rollback was successful
}
