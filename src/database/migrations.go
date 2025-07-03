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
	// Removed "Create whatsapp_messages table" so it will recreate
}

// GetMigrations returns only migrations that haven't been completed
func GetMigrations() []Migration {
	allMigrations := []Migration{
		{
			Name: "Fix whatsapp_chats missing columns and rename",
			SQL: `
			-- First add missing columns that might not exist
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS is_group BOOLEAN DEFAULT FALSE;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS is_muted BOOLEAN DEFAULT FALSE;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS last_message_text TEXT;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS last_message_time TIMESTAMP;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS unread_count INTEGER DEFAULT 0;
			ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS avatar_url TEXT;
			
			-- Now fix column name from 'name' to 'chat_name' if needed
			DO $$ 
			BEGIN
				-- Check if 'name' column exists and 'chat_name' doesn't
				IF EXISTS (SELECT 1 FROM information_schema.columns 
						   WHERE table_name = 'whatsapp_chats' AND column_name = 'name') 
				   AND NOT EXISTS (SELECT 1 FROM information_schema.columns 
								   WHERE table_name = 'whatsapp_chats' AND column_name = 'chat_name') THEN
					-- Rename 'name' to 'chat_name'
					ALTER TABLE whatsapp_chats RENAME COLUMN name TO chat_name;
				END IF;
				
				-- Ensure chat_name column exists (in case both are missing)
				ALTER TABLE whatsapp_chats ADD COLUMN IF NOT EXISTS chat_name VARCHAR(255);
			END $$;
			`,
		},
		{
			Name: "Recreate whatsapp_messages table with proper schema",
			SQL: `
			-- Drop the existing table if it exists
			DROP TABLE IF EXISTS whatsapp_messages CASCADE;
			
			-- Create table with proper structure
			CREATE TABLE whatsapp_messages (
				id SERIAL PRIMARY KEY,
				device_id VARCHAR(255) NOT NULL,
				chat_jid VARCHAR(255) NOT NULL,
				message_id VARCHAR(255) NOT NULL,
				sender_jid VARCHAR(255),
				sender_name VARCHAR(255),
				message_text TEXT,
				message_type VARCHAR(50) DEFAULT 'text',
				media_url TEXT,
				is_sent BOOLEAN DEFAULT FALSE,
				is_read BOOLEAN DEFAULT FALSE,
				timestamp BIGINT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(device_id, message_id)
			);

			-- Create indexes for performance
			CREATE INDEX idx_whatsapp_messages_device_chat ON whatsapp_messages(device_id, chat_jid);
			CREATE INDEX idx_whatsapp_messages_timestamp ON whatsapp_messages(timestamp DESC);

			-- Create function to validate and fix timestamps
			CREATE OR REPLACE FUNCTION fix_whatsapp_message_timestamp()
			RETURNS TRIGGER AS $$
			BEGIN
				-- If timestamp is in milliseconds (13+ digits), convert to seconds
				IF NEW.timestamp > 1000000000000 THEN
					NEW.timestamp := NEW.timestamp / 1000;
				END IF;
				
				-- If timestamp is more than 1 year in future, use current time
				IF NEW.timestamp > EXTRACT(EPOCH FROM NOW() + INTERVAL '1 year')::BIGINT THEN
					NEW.timestamp := EXTRACT(EPOCH FROM NOW())::BIGINT;
				END IF;
				
				-- If timestamp is negative or too small, use current time
				IF NEW.timestamp < 946684800 THEN -- Before year 2000
					NEW.timestamp := EXTRACT(EPOCH FROM NOW())::BIGINT;
				END IF;
				
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;

			-- Create trigger to fix timestamps automatically
			DROP TRIGGER IF EXISTS fix_timestamp_before_insert ON whatsapp_messages;
			CREATE TRIGGER fix_timestamp_before_insert
			BEFORE INSERT OR UPDATE ON whatsapp_messages
			FOR EACH ROW
			EXECUTE FUNCTION fix_whatsapp_message_timestamp();

			-- Function to keep only recent 20 messages per chat
			CREATE OR REPLACE FUNCTION limit_chat_messages() 
			RETURNS TRIGGER AS $$
			BEGIN
				DELETE FROM whatsapp_messages 
				WHERE device_id = NEW.device_id 
				AND chat_jid = NEW.chat_jid
				AND message_id NOT IN (
					SELECT message_id 
					FROM whatsapp_messages 
					WHERE device_id = NEW.device_id 
					AND chat_jid = NEW.chat_jid
					ORDER BY timestamp DESC 
					LIMIT 20
				);
				RETURN NEW;
			END;
			$$ LANGUAGE plpgsql;

			-- Apply message limit trigger
			DROP TRIGGER IF EXISTS limit_messages_trigger ON whatsapp_messages;
			CREATE TRIGGER limit_messages_trigger 
			AFTER INSERT ON whatsapp_messages 
			FOR EACH ROW EXECUTE FUNCTION limit_chat_messages();
			`,
		},
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
				ALTER TABLE broadcast_messages 
				ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ;
				
				-- Update scheduled_at from created_at for existing records
				UPDATE broadcast_messages 
				SET scheduled_at = created_at 
				WHERE scheduled_at IS NULL;
			`,
		},
		{
			Name: "Add updated_at to broadcast_messages",
			SQL: `
				ALTER TABLE broadcast_messages 
				ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
				
				-- Update existing records
				UPDATE broadcast_messages 
				SET updated_at = CURRENT_TIMESTAMP 
				WHERE updated_at IS NULL;
			`,
		},
		{
			Name: "Create time validation function",
			SQL: `
CREATE OR REPLACE FUNCTION is_valid_time(time_str TEXT) 
RETURNS BOOLEAN AS $$
BEGIN
    -- Check basic format HH:MM
    IF time_str !~ '^[0-2][0-9]:[0-5][0-9]$' THEN
        RETURN FALSE;
    END IF;
    
    -- Check hour is valid (00-23)
    IF CAST(SPLIT_PART(time_str, ':', 1) AS INTEGER) > 23 THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
EXCEPTION
    WHEN OTHERS THEN
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
		{
			Name: "Add sequence progress tracking",
			SQL: `
-- Add progress tracking fields to sequences table
ALTER TABLE sequences 
ADD COLUMN IF NOT EXISTS total_contacts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS active_contacts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS completed_contacts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS failed_contacts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS progress_percentage DECIMAL(5,2) DEFAULT 0.00,
ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS estimated_completion_at TIMESTAMP;

-- Add index for better performance
CREATE INDEX IF NOT EXISTS idx_sequences_progress ON sequences(progress_percentage);
CREATE INDEX IF NOT EXISTS idx_sequences_last_activity ON sequences(last_activity_at);

-- Add status column to sequence_contacts if missing
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active';
`,
		},
		{
			Name: "Add AI Campaign Feature",
			SQL: `
-- AI Campaign Feature Migration
-- This migration adds support for AI-powered lead management and campaign distribution

-- 1. Create leads_ai table for AI-managed leads
CREATE TABLE IF NOT EXISTS leads_ai (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    device_id UUID, -- Initially NULL, assigned during campaign
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50) NOT NULL,
    email VARCHAR(255),
    niche VARCHAR(255),
    source VARCHAR(255) DEFAULT 'ai_manual',
    status VARCHAR(50) DEFAULT 'pending', -- pending, assigned, sent, failed
    target_status VARCHAR(50) DEFAULT 'prospect', -- prospect/customer
    notes TEXT,
    assigned_at TIMESTAMP, -- When assigned to device
    sent_at TIMESTAMP, -- When message was sent
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES user_devices(id) ON DELETE SET NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_leads_ai_user_id ON leads_ai(user_id);
CREATE INDEX IF NOT EXISTS idx_leads_ai_device_id ON leads_ai(device_id);
CREATE INDEX IF NOT EXISTS idx_leads_ai_status ON leads_ai(status);
CREATE INDEX IF NOT EXISTS idx_leads_ai_niche ON leads_ai(niche);
CREATE INDEX IF NOT EXISTS idx_leads_ai_phone ON leads_ai(phone);

-- 2. Add AI columns to campaigns table
ALTER TABLE campaigns 
ADD COLUMN IF NOT EXISTS ai VARCHAR(10),
ADD COLUMN IF NOT EXISTS "limit" INTEGER DEFAULT 0;

-- 3. Create ai_campaign_progress table for tracking device usage
CREATE TABLE IF NOT EXISTS ai_campaign_progress (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL,
    device_id UUID NOT NULL,
    leads_sent INTEGER DEFAULT 0,
    leads_failed INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active', -- active, limit_reached, failed
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES user_devices(id) ON DELETE CASCADE,
    UNIQUE(campaign_id, device_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_ai_campaign_progress_campaign_id ON ai_campaign_progress(campaign_id);
CREATE INDEX IF NOT EXISTS idx_ai_campaign_progress_device_id ON ai_campaign_progress(device_id);
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