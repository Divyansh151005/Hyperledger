package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Database wraps a PostgreSQL connection
type Database struct {
	conn *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(databaseURL string) (*Database, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &Database{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.conn.Close()
}

// GetConnection returns the underlying database connection
func (d *Database) GetConnection() *sql.DB {
	return d.conn
}

// initSchema initializes the database schema
func (d *Database) initSchema() error {
	queries := []string{
		// Identity mappings table
		`CREATE TABLE IF NOT EXISTS identity_mappings (
			certificate_id VARCHAR(255) PRIMARY KEY,
			msp_id VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			user_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Audit logs table
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id SERIAL PRIMARY KEY,
			event_type VARCHAR(100) NOT NULL,
			chaincode_name VARCHAR(100) NOT NULL,
			transaction_id VARCHAR(255) NOT NULL,
			block_number BIGINT NOT NULL,
			invoker_msp VARCHAR(100),
			client_id VARCHAR(255),
			payload JSONB,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Medical records metadata table (for quick lookups)
		`CREATE TABLE IF NOT EXISTS medical_records_metadata (
			record_id VARCHAR(255) PRIMARY KEY,
			patient_id VARCHAR(255) NOT NULL,
			hospital_id VARCHAR(100) NOT NULL,
			record_hash VARCHAR(64) NOT NULL,
			storage_location TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Encryption keys table (keys encrypted at rest)
		`CREATE TABLE IF NOT EXISTS encryption_keys (
			authorization_request_id VARCHAR(255) PRIMARY KEY,
			encrypted_key BYTEA NOT NULL,
			key_metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP
		)`,
		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_identity_mappings_msp_id ON identity_mappings(msp_id)`,
		`CREATE INDEX IF NOT EXISTS idx_identity_mappings_role ON identity_mappings(role)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_medical_records_patient_id ON medical_records_metadata(patient_id)`,
		`CREATE INDEX IF NOT EXISTS idx_medical_records_hospital_id ON medical_records_metadata(hospital_id)`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}
