package repository

import (
	"database/sql"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/database"
	"github.com/sirupsen/logrus"
)

// InitSequenceSchema ensures the sequences table has all required columns
func InitSequenceSchema() error {
	db := database.GetDB()
	
	// Add status column if it doesn't exist
	_, err := db.Exec(`
		ALTER TABLE sequences 
		ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'inactive'
	`)
	if err != nil {
		logrus.Warnf("Failed to add status column (might already exist): %v", err)
	}
	
	// Update existing sequences to have proper status based on is_active
	_, err = db.Exec(`
		UPDATE sequences 
		SET status = CASE 
			WHEN is_active = true THEN 'active' 
			ELSE 'inactive' 
		END
		WHERE status IS NULL OR status = ''
	`)
	if err != nil {
		logrus.Warnf("Failed to update sequence statuses: %v", err)
	}
	
	// Add other missing columns
	alterCommands := []string{
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'all'",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS schedule_time VARCHAR(5) DEFAULT '09:00'",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS min_delay_seconds INTEGER DEFAULT 30",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS max_delay_seconds INTEGER DEFAULT 60",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS contacts_count INTEGER DEFAULT 0",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS total_contacts INTEGER DEFAULT 0",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS active_contacts INTEGER DEFAULT 0",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS completed_contacts INTEGER DEFAULT 0",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS failed_contacts INTEGER DEFAULT 0",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS progress_percentage DECIMAL(5,2) DEFAULT 0.00",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMP",
		"ALTER TABLE sequences ADD COLUMN IF NOT EXISTS estimated_completion_at TIMESTAMP",
	}
	
	for _, cmd := range alterCommands {
		if _, err := db.Exec(cmd); err != nil {
			logrus.Debugf("Column might already exist: %v", err)
		}
	}
	
	logrus.Info("Sequence schema initialized successfully")
	return nil
}
