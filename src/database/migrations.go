package database

import (
	"time"
)

// Migrations that have already been applied
// Add completed migrations here to skip them
var completedMigrations = map[string]bool{
	"Add target_status columns":             true,
	"Add time_schedule columns":             true,
	"Add scheduled_at for timezone support": true,
	"Add updated_at to broadcast_messages":  true,
	"Create time validation function":       true,
	"Fix leads table columns":               true,
	"Fix whatsmeow_message_secrets table":   true,
}

// GetMigrations returns only migrations that haven't been completed
func GetMigrations() []Migration {
	allMigrations := []Migration{
		{
			Name: "Add target_status columns",
			SQL: `
				ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'all';
				ALTER TABLE sequences ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'all';
				UPDATE campaigns SET target_status = 'all' WHERE target_status IS NULL;
				UPDATE sequences SET target_status = 'all' WHERE target_status IS NULL;
			`,
		},
		{
			Name: "Add time_schedule columns",
			SQL: `
				-- Add time_schedule to campaigns
				ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS time_schedule TEXT;
				UPDATE campaigns SET time_schedule = scheduled_time WHERE time_schedule IS NULL AND scheduled_time IS NOT NULL;
				
				-- Add time_schedule to sequences
				ALTER TABLE sequences ADD COLUMN IF NOT EXISTS time_schedule TEXT;
				UPDATE sequences SET time_schedule = schedule_time WHERE time_schedule IS NULL AND schedule_time IS NOT NULL;
				
				-- Add time_schedule to sequence_steps
				ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS time_schedule TEXT;
				UPDATE sequence_steps SET time_schedule = schedule_time WHERE time_schedule IS NULL AND schedule_time IS NOT NULL;
			`,
		},
		{
			Name: "Add scheduled_at for timezone support",
			SQL: `
				-- Add TIMESTAMPTZ column for proper timezone support
				ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
				
				-- Migrate existing data to TIMESTAMPTZ (assuming Malaysia timezone)
				UPDATE campaigns 
				SET scheduled_at = 
					CASE 
						WHEN time_schedule IS NULL OR time_schedule = '' THEN 
							campaign_date::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur'
						ELSE 
							(campaign_date || ' ' || time_schedule)::TIMESTAMP AT TIME ZONE 'Asia/Kuala_Lumpur'
					END
				WHERE scheduled_at IS NULL;
				
				-- Create optimized indexes
				CREATE INDEX IF NOT EXISTS idx_campaigns_scheduled_at ON campaigns(scheduled_at) WHERE status = 'pending';
				CREATE INDEX IF NOT EXISTS idx_campaigns_scheduled_at_status ON campaigns(scheduled_at, status);
			`,
		},
		{
			Name: "Add updated_at to broadcast_messages",
			SQL: `
				ALTER TABLE broadcast_messages ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
			`,
		},
		{
			Name: "Create time validation function",
			SQL: `
				CREATE OR REPLACE FUNCTION is_valid_time_schedule(time_str TEXT) 
				RETURNS BOOLEAN AS $$
				BEGIN
					IF time_str IS NULL OR time_str = '' THEN
						RETURN TRUE;
					END IF;
					
					IF time_str ~ '^\d{2}:\d{2}(:\d{2})?$' THEN
						BEGIN
							PERFORM time_str::TIME;
							RETURN TRUE;
						EXCEPTION WHEN OTHERS THEN
							RETURN FALSE;
						END;
					END IF;
					
					RETURN FALSE;
				END;
				$$ LANGUAGE plpgsql;
			`,
		},
		{
			Name: "Fix leads table columns",
			SQL: `
				-- Ensure target_status exists in leads table
				ALTER TABLE leads ADD COLUMN IF NOT EXISTS target_status VARCHAR(50) DEFAULT 'prospect';
				UPDATE leads SET target_status = 'prospect' WHERE target_status IS NULL;
			`,
		},
		{
			Name: "Fix whatsmeow_message_secrets table",
			SQL: `
				-- Create whatsmeow_message_secrets table if not exists
				CREATE TABLE IF NOT EXISTS whatsmeow_message_secrets (
					our_jid text,
					chat_jid text,
					sender_jid text,
					message_id text,
					key bytea,
					PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id)
				);
				
				-- Add key column if missing
				DO $$ 
				BEGIN
					IF NOT EXISTS (
						SELECT 1 
						FROM information_schema.columns 
						WHERE table_name = 'whatsmeow_message_secrets' 
						AND column_name = 'key'
					) THEN
						ALTER TABLE whatsmeow_message_secrets 
						ADD COLUMN key bytea;
					END IF;
				END $$;
			`,
		},
	}
	
	// Filter out completed migrations
	var pendingMigrations []Migration
	for _, m := range allMigrations {
		if !completedMigrations[m.Name] {
			pendingMigrations = append(pendingMigrations, m)
		}
	}
	
	return pendingMigrations
}

// Migration represents a database migration
type Migration struct {
	Name string
	SQL  string
	RunAt time.Time
}