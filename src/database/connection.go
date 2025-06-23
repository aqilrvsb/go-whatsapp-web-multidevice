package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
	
	_ "github.com/lib/pq"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
)

var (
	db   *sql.DB
	once sync.Once
)

// GetDB returns the database connection
func GetDB() *sql.DB {
	once.Do(func() {
		var err error
		
		// Parse PostgreSQL connection string
		db, err = sql.Open("postgres", config.DBURI)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		
		// Configure connection pool for 200+ users
		db.SetMaxOpenConns(100)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(time.Hour)
		
		// Test connection
		if err := db.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
		
		// Initialize schema
		if err := InitializeSchema(); err != nil {
			log.Fatalf("Failed to initialize schema: %v", err)
		}
		
		log.Println("Database connection established")
	})
	
	return db
}
// InitializeSchema creates tables if they don't exist
func InitializeSchema() error {
	schema := `
	-- Create extension for UUID generation if not exists
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Create users table
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		full_name VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login TIMESTAMP
	);

	-- Create user_devices table
	CREATE TABLE IF NOT EXISTS user_devices (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		device_name VARCHAR(255) NOT NULL,
		phone VARCHAR(50),
		jid VARCHAR(255),
		status VARCHAR(50) DEFAULT 'offline',
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, jid)
	);

	-- Create user_sessions table
	CREATE TABLE IF NOT EXISTS user_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token VARCHAR(255) UNIQUE NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create message_analytics table
	CREATE TABLE IF NOT EXISTS message_analytics (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		device_id UUID REFERENCES user_devices(id) ON DELETE SET NULL,
		message_id VARCHAR(255) NOT NULL,
		jid VARCHAR(255) NOT NULL,
		content TEXT,
		is_from_me BOOLEAN DEFAULT false,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id)
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_token ON user_sessions(token);
	CREATE INDEX IF NOT EXISTS idx_message_analytics_user_id ON message_analytics(user_id);
	CREATE INDEX IF NOT EXISTS idx_message_analytics_created_at ON message_analytics(created_at);
	
	-- Create default admin user if not exists
	INSERT INTO users (email, full_name, password_hash, is_active) 
	VALUES ('admin@whatsapp.com', 'Administrator', '$2a$10$K.0HwpsoPDGaB/atFBmmXOGTw4ceeg33.WrxJx/sU1l44onhKi9am', true)
	ON CONFLICT (email) DO NOTHING;
	`
	
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	// Run cleanup for expired sessions
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			db.Exec("DELETE FROM user_sessions WHERE expires_at < CURRENT_TIMESTAMP")
		}
	}()
	
	return nil
}