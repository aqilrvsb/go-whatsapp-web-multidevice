-- Optimized Sequence System for 3000 Devices (No Retry Version)
-- This migration adds support for individual flow tracking with single attempt only

-- 1. Add sequence_stepid column to sequence_contacts if missing
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS sequence_stepid UUID REFERENCES sequence_steps(id) ON DELETE SET NULL;

-- 2. Add processing_device_id to track which device is handling each flow
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS processing_device_id UUID REFERENCES user_devices(id) ON DELETE SET NULL;

-- 3. Add processing_started_at to detect stuck processing
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMP;

-- 4. Add created_at for tracking when flow record was created
ALTER TABLE sequence_contacts 
ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- 5. Create indexes for optimal performance with 3000 devices
CREATE INDEX IF NOT EXISTS idx_sc_sequence_stepid ON sequence_contacts(sequence_stepid);
CREATE INDEX IF NOT EXISTS idx_sc_processing_device ON sequence_contacts(processing_device_id) 
WHERE processing_device_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sc_active_ready ON sequence_contacts(status, next_trigger_time) 
WHERE status = 'active' AND processing_device_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_sc_phone_sequence ON sequence_contacts(contact_phone, sequence_id);

-- 6. Update the unique constraint to allow multiple flow records per contact
-- First drop the old constraint if it exists
ALTER TABLE sequence_contacts 
DROP CONSTRAINT IF EXISTS uq_sequence_contact;

-- Add new constraint that allows multiple records per contact (one per flow)
CREATE UNIQUE INDEX IF NOT EXISTS idx_sc_unique_flow 
ON sequence_contacts(sequence_id, contact_phone, sequence_stepid);

-- 7. Create device load balance table if not exists
CREATE TABLE IF NOT EXISTS device_load_balance (
    device_id UUID PRIMARY KEY REFERENCES user_devices(id) ON DELETE CASCADE,
    messages_hour INTEGER DEFAULT 0,
    messages_today INTEGER DEFAULT 0,
    last_reset_hour TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_reset_day TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_available BOOLEAN DEFAULT true,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 8. Create function to reset device counters
CREATE OR REPLACE FUNCTION reset_device_counters() RETURNS void AS $$
BEGIN
    -- Reset hourly counters
    UPDATE device_load_balance
    SET messages_hour = 0, 
        last_reset_hour = CURRENT_TIMESTAMP
    WHERE last_reset_hour < CURRENT_TIMESTAMP - INTERVAL '1 hour';
    
    -- Reset daily counters
    UPDATE device_load_balance
    SET messages_today = 0, 
        last_reset_day = CURRENT_TIMESTAMP
    WHERE last_reset_day < CURRENT_TIMESTAMP - INTERVAL '1 day';
END;
$$ LANGUAGE plpgsql;

-- 9. Create trigger to auto-reset counters
CREATE OR REPLACE FUNCTION check_and_reset_counters() RETURNS trigger AS $$
BEGIN
    PERFORM reset_device_counters();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if exists
DROP TRIGGER IF EXISTS device_counter_reset ON device_load_balance;

-- Create trigger to run on any update
CREATE TRIGGER device_counter_reset
BEFORE UPDATE ON device_load_balance
FOR EACH ROW
EXECUTE FUNCTION check_and_reset_counters();

-- 10. Initialize device load balance for existing devices
INSERT INTO device_load_balance (device_id)
SELECT id FROM user_devices
ON CONFLICT (device_id) DO NOTHING;

-- 11. Add schedule_time column to sequences if missing
ALTER TABLE sequences 
ADD COLUMN IF NOT EXISTS schedule_time VARCHAR(5); -- Format: "HH:MM"

-- 12. Create view for monitoring sequence progress (no retry count)
CREATE OR REPLACE VIEW sequence_progress_monitor AS
SELECT 
    s.id as sequence_id,
    s.name as sequence_name,
    s.trigger,
    s.status as sequence_status,
    s.is_active,
    COUNT(DISTINCT sc.contact_phone) as total_contacts,
    COUNT(DISTINCT sc.contact_phone) FILTER (WHERE sc.status = 'active') as active_contacts,
    COUNT(DISTINCT sc.contact_phone) FILTER (WHERE sc.status = 'pending') as pending_contacts,
    COUNT(DISTINCT sc.contact_phone) FILTER (WHERE sc.status = 'sent') as sent_contacts,
    COUNT(DISTINCT sc.contact_phone) FILTER (WHERE sc.status = 'failed') as failed_contacts,
    COUNT(DISTINCT sc.contact_phone) FILTER (WHERE sc.status = 'completed') as completed_contacts,
    COUNT(DISTINCT ss.id) as total_steps,
    COUNT(DISTINCT sc.sequence_stepid) FILTER (WHERE sc.status = 'sent') as steps_sent,
    COUNT(DISTINCT sc.sequence_stepid) FILTER (WHERE sc.status = 'failed') as steps_failed
FROM sequences s
LEFT JOIN sequence_contacts sc ON sc.sequence_id = s.id
LEFT JOIN sequence_steps ss ON ss.sequence_id = s.id
GROUP BY s.id, s.name, s.trigger, s.status, s.is_active;

-- 13. Create view for device performance monitoring
CREATE OR REPLACE VIEW device_performance_monitor AS
SELECT 
    d.id as device_id,
    d.name as device_name,
    d.phone as device_phone,
    d.status as device_status,
    COALESCE(dlb.messages_hour, 0) as messages_hour,
    COALESCE(dlb.messages_today, 0) as messages_today,
    COALESCE(dlb.is_available, true) as is_available,
    COUNT(DISTINCT sc.id) as current_processing,
    COUNT(DISTINCT sc.id) FILTER (WHERE sc.status = 'sent' AND sc.completed_at > NOW() - INTERVAL '1 hour') as sent_last_hour,
    COUNT(DISTINCT sc.id) FILTER (WHERE sc.status = 'failed' AND sc.completed_at > NOW() - INTERVAL '1 hour') as failed_last_hour
FROM user_devices d
LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
LEFT JOIN sequence_contacts sc ON sc.processing_device_id = d.id 
    AND sc.processing_started_at > NOW() - INTERVAL '5 minutes'
GROUP BY d.id, d.name, d.phone, d.status, dlb.messages_hour, dlb.messages_today, dlb.is_available;

-- 14. Create function to get next available device with load balancing
CREATE OR REPLACE FUNCTION get_next_available_device(
    p_preferred_device_id UUID DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_device_id UUID;
BEGIN
    -- Try preferred device first if available and not overloaded
    IF p_preferred_device_id IS NOT NULL THEN
        SELECT d.id INTO v_device_id
        FROM user_devices d
        LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
        WHERE d.id = p_preferred_device_id
            AND d.status = 'online'
            AND COALESCE(dlb.is_available, true) = true
            AND COALESCE(dlb.messages_hour, 0) < 50  -- Lower threshold for preferred device
            AND COALESCE(dlb.messages_today, 0) < 800;
        
        IF v_device_id IS NOT NULL THEN
            RETURN v_device_id;
        END IF;
    END IF;
    
    -- Get least loaded available device
    SELECT d.id INTO v_device_id
    FROM user_devices d
    LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
    LEFT JOIN (
        SELECT processing_device_id, COUNT(*) as active_count
        FROM sequence_contacts
        WHERE processing_device_id IS NOT NULL
            AND processing_started_at > NOW() - INTERVAL '5 minutes'
        GROUP BY processing_device_id
    ) ac ON ac.processing_device_id = d.id
    WHERE d.status = 'online'
        AND COALESCE(dlb.is_available, true) = true
        AND COALESCE(dlb.messages_hour, 0) < 80
        AND COALESCE(dlb.messages_today, 0) < 800
    ORDER BY 
        -- Score calculation: hourly messages weighted 70%, current processing 30%
        COALESCE(dlb.messages_hour, 0) * 0.7 + COALESCE(ac.active_count, 0) * 0.3 ASC
    LIMIT 1;
    
    RETURN v_device_id;
END;
$$ LANGUAGE plpgsql;

-- 15. Create function to update device load after sending message
CREATE OR REPLACE FUNCTION update_device_load(
    p_device_id UUID
) RETURNS void AS $$
BEGIN
    -- Reset counters if needed
    PERFORM reset_device_counters();
    
    -- Update counters
    INSERT INTO device_load_balance (device_id, messages_hour, messages_today, updated_at)
    VALUES (p_device_id, 1, 1, CURRENT_TIMESTAMP)
    ON CONFLICT (device_id) DO UPDATE
    SET messages_hour = device_load_balance.messages_hour + 1,
        messages_today = device_load_balance.messages_today + 1,
        updated_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;

-- 16. Add trigger to auto-update sequence progress
CREATE OR REPLACE FUNCTION update_sequence_progress() RETURNS trigger AS $$
BEGIN
    -- Update sequence progress stats
    UPDATE sequences s
    SET total_contacts = sq.total_contacts,
        active_contacts = sq.active_contacts,
        completed_contacts = sq.completed_contacts,
        failed_contacts = sq.failed_contacts,
        progress_percentage = CASE 
            WHEN sq.total_contacts > 0 
            THEN ((sq.completed_contacts + sq.failed_contacts)::DECIMAL / sq.total_contacts::DECIMAL) * 100
            ELSE 0 
        END,
        last_activity_at = CURRENT_TIMESTAMP
    FROM (
        SELECT 
            sequence_id,
            COUNT(DISTINCT contact_phone) as total_contacts,
            COUNT(DISTINCT contact_phone) FILTER (WHERE status IN ('active', 'pending')) as active_contacts,
            COUNT(DISTINCT contact_phone) FILTER (WHERE status = 'completed') as completed_contacts,
            COUNT(DISTINCT contact_phone) FILTER (WHERE status = 'failed') as failed_contacts
        FROM sequence_contacts
        WHERE sequence_id = COALESCE(NEW.sequence_id, OLD.sequence_id)
        GROUP BY sequence_id
    ) sq
    WHERE s.id = sq.sequence_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if exists
DROP TRIGGER IF EXISTS sequence_progress_update ON sequence_contacts;

-- Create trigger
CREATE TRIGGER sequence_progress_update
AFTER INSERT OR UPDATE OR DELETE ON sequence_contacts
FOR EACH ROW
EXECUTE FUNCTION update_sequence_progress();

-- 17. Add partial indexes for common queries
CREATE INDEX IF NOT EXISTS idx_sequences_active_scheduled 
ON sequences(schedule_time) 
WHERE is_active = true AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_leads_with_triggers 
ON leads(user_id, phone) 
WHERE trigger IS NOT NULL AND trigger != '';

-- 18. Create function to mark stuck processing as failed (no retry)
CREATE OR REPLACE FUNCTION cleanup_stuck_processing() RETURNS void AS $$
BEGIN
    -- Mark contacts stuck in processing for more than 5 minutes as failed
    UPDATE sequence_contacts
    SET processing_device_id = NULL,
        processing_started_at = NULL,
        status = 'failed',
        completed_at = CURRENT_TIMESTAMP
    WHERE processing_device_id IS NOT NULL
        AND processing_started_at < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
        AND status = 'active';
END;
$$ LANGUAGE plpgsql;

-- 19. Create monitoring view for failed flows
CREATE OR REPLACE VIEW failed_flows_monitor AS
SELECT 
    sc.id as flow_id,
    sc.contact_phone,
    sc.contact_name,
    s.name as sequence_name,
    ss.day_number,
    ss.content as message_content,
    sc.completed_at as failed_at,
    sc.processing_device_id as last_device_id,
    d.name as last_device_name
FROM sequence_contacts sc
JOIN sequences s ON s.id = sc.sequence_id
JOIN sequence_steps ss ON ss.id = sc.sequence_stepid
LEFT JOIN user_devices d ON d.id = sc.processing_device_id
WHERE sc.status = 'failed'
ORDER BY sc.completed_at DESC;

-- 20. Add comment documentation
COMMENT ON TABLE sequence_contacts IS 'Tracks individual flow progress for each contact in sequences. Each flow/step gets its own record. No retry - single attempt only.';
COMMENT ON COLUMN sequence_contacts.sequence_stepid IS 'References the specific step being processed';
COMMENT ON COLUMN sequence_contacts.processing_device_id IS 'Device currently processing this flow';
COMMENT ON COLUMN device_load_balance.messages_hour IS 'Messages sent in current hour (auto-resets)';
COMMENT ON COLUMN device_load_balance.messages_today IS 'Messages sent today (auto-resets at midnight)';

-- 21. Create scheduled job to clean stuck processing (run every 5 minutes)
-- Note: This is a placeholder - actual scheduling depends on your system
-- You might use pg_cron, external scheduler, or application-level scheduling
-- Example with pg_cron (if installed):
-- SELECT cron.schedule('cleanup-stuck-sequences', '*/5 * * * *', 'SELECT cleanup_stuck_processing();');

-- Migration complete!
-- This schema supports:
-- - Individual flow tracking per contact
-- - 3000 device load balancing
-- - Single attempt only (no retry)
-- - Automatic progress updates
-- - Schedule time respecting
-- - Performance monitoring