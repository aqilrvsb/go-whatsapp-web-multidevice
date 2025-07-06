package database

import (
	"github.com/sirupsen/logrus"
)

// AddSequenceTriggerOptimization adds trigger-based sequence system for 3000 devices
func AddSequenceTriggerOptimization() Migration {
	return Migration{
		Name: "Add Sequence Trigger Optimization",
		SQL: `
-- ============================================
-- Simple Trigger System for Leads
-- ============================================

-- Add trigger column to leads table (comma-separated like niche)
ALTER TABLE leads ADD COLUMN IF NOT EXISTS trigger VARCHAR(500);

-- Create index for trigger searches
CREATE INDEX IF NOT EXISTS idx_leads_trigger ON leads(trigger) WHERE trigger IS NOT NULL AND trigger != '';

-- ============================================
-- Enhance Sequence Steps
-- ============================================

-- Ensure sequence_steps has proper trigger flow
ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS next_trigger VARCHAR(255);
ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS trigger_delay_hours INT DEFAULT 24;
ALTER TABLE sequence_steps ADD COLUMN IF NOT EXISTS is_final_step BOOLEAN DEFAULT false;

-- Create trigger lookup index
CREATE INDEX IF NOT EXISTS idx_sequence_steps_trigger ON sequence_steps(trigger);

-- Mark final steps in sequences
UPDATE sequence_steps ss
SET is_final_step = true
WHERE NOT EXISTS (
    SELECT 1 FROM sequence_steps ss2 
    WHERE ss2.sequence_id = ss.sequence_id 
    AND ss2.day_number > ss.day_number
);

-- ============================================
-- Sequence Contacts Optimization
-- ============================================

-- Add processing fields for better tracking
ALTER TABLE sequence_contacts ADD COLUMN IF NOT EXISTS last_trigger_sent VARCHAR(255);
ALTER TABLE sequence_contacts ADD COLUMN IF NOT EXISTS next_trigger_time TIMESTAMP;
ALTER TABLE sequence_contacts ADD COLUMN IF NOT EXISTS processing_device_id UUID;

-- Index for fast queries
CREATE INDEX IF NOT EXISTS idx_sequence_contacts_ready 
ON sequence_contacts(status, last_message_at) 
WHERE status = 'active';

-- ============================================
-- Helper Functions for Trigger Management
-- ============================================

-- Function to add trigger to lead
CREATE OR REPLACE FUNCTION add_trigger_to_lead(
    p_lead_id UUID,
    p_new_trigger VARCHAR(255)
) RETURNS BOOLEAN AS $$
BEGIN
    UPDATE leads
    SET trigger = CASE
        WHEN trigger IS NULL OR trigger = '' THEN p_new_trigger
        WHEN trigger LIKE '%' || p_new_trigger || '%' THEN trigger
        ELSE trigger || ',' || p_new_trigger
    END
    WHERE id = p_lead_id;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function to remove trigger from lead
CREATE OR REPLACE FUNCTION remove_trigger_from_lead(
    p_lead_id UUID,
    p_trigger_to_remove VARCHAR(255)
) RETURNS BOOLEAN AS $$
DECLARE
    v_old_trigger TEXT;
    v_new_trigger TEXT;
BEGIN
    -- Get current trigger
    SELECT trigger INTO v_old_trigger FROM leads WHERE id = p_lead_id;
    
    IF v_old_trigger IS NULL THEN
        RETURN true;
    END IF;
    
    -- Remove the trigger
    v_new_trigger := TRIM(BOTH ',' FROM
        REPLACE(',' || v_old_trigger || ',', ',' || p_trigger_to_remove || ',', ',')
    );
    
    -- Update lead
    UPDATE leads 
    SET trigger = NULLIF(v_new_trigger, '')
    WHERE id = p_lead_id;
    
    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Device Message Queue for 3000 devices
-- ============================================

CREATE TABLE IF NOT EXISTS device_message_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL,
    sequence_contact_id UUID,
    phone VARCHAR(20) NOT NULL,
    message_type VARCHAR(50),
    message_text TEXT,
    media_url TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    priority INT DEFAULT 5,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP,
    error_message TEXT,
    retry_count INT DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_queue_device_pending 
ON device_message_queue(device_id, status, priority DESC, created_at ASC)
WHERE status = 'pending';

-- ============================================
-- Performance Tracking
-- ============================================

CREATE TABLE IF NOT EXISTS device_performance (
    device_id UUID PRIMARY KEY,
    messages_sent_hour INT DEFAULT 0,
    messages_sent_today INT DEFAULT 0,
    last_hour_reset TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_day_reset TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    success_rate INT DEFAULT 100,
    is_healthy BOOLEAN DEFAULT true,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- View for monitoring
CREATE OR REPLACE VIEW sequence_performance AS
SELECT 
    s.name as sequence_name,
    COUNT(DISTINCT sc.id) as total_contacts,
    COUNT(CASE WHEN sc.status = 'active' THEN 1 END) as active,
    COUNT(CASE WHEN sc.status = 'completed' THEN 1 END) as completed,
    AVG(sc.current_day)::INT as avg_day
FROM sequences s
LEFT JOIN sequence_contacts sc ON sc.sequence_id = s.id
WHERE s.is_active = true
GROUP BY s.id, s.name;
		`,
	}
}

// AddTriggerProcessingFunction adds the main processing function
func AddTriggerProcessingFunction() Migration {
	return Migration{
		Name: "Add Trigger Processing Function",
		SQL: `
-- Main processing function for sequence triggers
CREATE OR REPLACE FUNCTION process_sequence_triggers(
    p_limit INT DEFAULT 1000
) RETURNS TABLE(
    processed_count INT,
    device_id UUID,
    messages_queued INT
) AS $$
DECLARE
    v_total_processed INT := 0;
    v_contact RECORD;
    v_device_id UUID;
    v_device_count INT := 0;
    v_messages_per_device INT;
BEGIN
    -- Get count of healthy devices
    SELECT COUNT(*) INTO v_device_count
    FROM user_devices ud
    JOIN device_performance dp ON dp.device_id = ud.id
    WHERE ud.status = 'online' 
    AND dp.is_healthy = true
    AND dp.messages_sent_hour < 80;
    
    IF v_device_count = 0 THEN
        RETURN;
    END IF;
    
    -- Calculate distribution
    v_messages_per_device := GREATEST(1, p_limit / v_device_count);
    
    -- Process contacts that are ready
    FOR v_contact IN
        SELECT 
            sc.id,
            sc.contact_phone,
            sc.sequence_id,
            sc.current_day,
            ss.message_text,
            ss.message_type,
            ss.media_url,
            ss.trigger,
            ss.next_trigger,
            ss.is_final_step,
            s.trigger_prefix,
            COALESCE(l.device_id, 
                (SELECT ud.id FROM user_devices ud 
                 WHERE ud.user_id = s.user_id 
                 AND ud.status = 'online' 
                 ORDER BY RANDOM() 
                 LIMIT 1)
            ) as assigned_device_id
        FROM sequence_contacts sc
        JOIN sequences s ON s.id = sc.sequence_id
        JOIN sequence_steps ss ON ss.sequence_id = sc.sequence_id 
            AND ss.day_number = (sc.current_day + 1)
        LEFT JOIN leads l ON l.phone = sc.contact_phone
        WHERE sc.status = 'active'
            AND s.is_active = true
            AND (sc.last_message_at IS NULL 
                 OR sc.last_message_at < NOW() - INTERVAL '24 hours')
        ORDER BY s.priority DESC, sc.last_message_at ASC
        LIMIT p_limit
    LOOP
        -- Queue the message
        INSERT INTO device_message_queue (
            device_id,
            sequence_contact_id,
            phone,
            message_type,
            message_text,
            media_url,
            priority
        ) VALUES (
            v_contact.assigned_device_id,
            v_contact.id,
            v_contact.contact_phone,
            v_contact.message_type,
            v_contact.message_text,
            v_contact.media_url,
            5
        );
        
        -- Update sequence contact
        UPDATE sequence_contacts
        SET 
            current_day = current_day + 1,
            last_message_at = CURRENT_TIMESTAMP,
            last_trigger_sent = v_contact.trigger,
            status = CASE 
                WHEN v_contact.is_final_step THEN 'completed'
                ELSE 'active'
            END,
            completed_at = CASE
                WHEN v_contact.is_final_step THEN CURRENT_TIMESTAMP
                ELSE NULL
            END
        WHERE id = v_contact.id;
        
        -- Update lead trigger if sequence completed
        IF v_contact.is_final_step AND v_contact.trigger_prefix IS NOT NULL THEN
            PERFORM remove_trigger_from_lead(
                (SELECT id FROM leads WHERE phone = v_contact.contact_phone LIMIT 1),
                v_contact.trigger_prefix || 'START'
            );
        END IF;
        
        v_total_processed := v_total_processed + 1;
    END LOOP;
    
    -- Return summary
    RETURN QUERY
    SELECT 
        v_total_processed,
        dmq.device_id,
        COUNT(*)::INT as msgs
    FROM device_message_queue dmq
    WHERE dmq.created_at >= NOW() - INTERVAL '1 minute'
    GROUP BY dmq.device_id;
END;
$$ LANGUAGE plpgsql;
		`,
	}
}
