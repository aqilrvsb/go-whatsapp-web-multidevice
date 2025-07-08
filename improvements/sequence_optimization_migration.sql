 sc ON sc.processing_device_id = d.id 
    AND sc.processing_started_at > NOW() - INTERVAL '5 minutes'
GROUP BY d.id, d.name, d.phone, d.status, dlb.messages_hour, dlb.messages_today, dlb.is_available;

-- 15. Create function to get next available device with load balancing
CREATE OR REPLACE FUNCTION get_next_available_device(
    p_preferred_device_id UUID DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_device_id UUID;
BEGIN
    -- Try preferred device first if available
    IF p_preferred_device_id IS NOT NULL THEN
        SELECT d.id INTO v_device_id
        FROM user_devices d
        LEFT JOIN device_load_balance dlb ON dlb.device_id = d.id
        WHERE d.id = p_preferred_device_id
            AND d.status = 'online'
            AND COALESCE(dlb.is_available, true) = true
            AND COALESCE(dlb.messages_hour, 0) < 80
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
        COALESCE(dlb.messages_hour, 0) * 0.7 + COALESCE(ac.active_count, 0) * 0.3 ASC
    LIMIT 1;
    
    RETURN v_device_id;
END;
$$ LANGUAGE plpgsql;

-- 16. Create function to update device load after sending message
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

-- 17. Add trigger to auto-update sequence progress
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
            COUNT(DISTINCT contact_phone) FILTER (WHERE status = 'active') as active_contacts,
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

-- 18. Performance optimization for 3000 devices
-- Increase work_mem for better sorting performance
-- Note: This should be set at session level in the application
-- SET work_mem = '256MB';

-- 19. Add partial indexes for common queries
CREATE INDEX IF NOT EXISTS idx_sequences_active_scheduled 
ON sequences(schedule_time) 
WHERE is_active = true AND status = 'active';

CREATE INDEX IF NOT EXISTS idx_leads_with_triggers 
ON leads(user_id, phone) 
WHERE trigger IS NOT NULL AND trigger != '';

-- 20. Create materialized view for faster sequence matching (refresh periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS sequence_lead_matches AS
SELECT 
    l.id as lead_id,
    l.phone,
    l.name,
    l.device_id as preferred_device_id,
    s.id as sequence_id,
    s.trigger as sequence_trigger,
    ss.id as step_id,
    ss.trigger as step_trigger,
    ss.is_entry_point
FROM leads l
CROSS JOIN sequences s
INNER JOIN sequence_steps ss ON ss.sequence_id = s.id
WHERE l.trigger IS NOT NULL 
    AND l.trigger != ''
    AND s.is_active = true
    AND s.status = 'active'
    AND ss.is_entry_point = true
    AND position(ss.trigger in l.trigger) > 0
    AND NOT EXISTS (
        SELECT 1 FROM sequence_contacts sc
        WHERE sc.sequence_id = s.id 
        AND sc.contact_phone = l.phone
    );

-- Create index on materialized view
CREATE INDEX IF NOT EXISTS idx_seq_lead_matches_phone 
ON sequence_lead_matches(phone, sequence_id);

-- 21. Add comment documentation
COMMENT ON TABLE sequence_contacts IS 'Tracks individual flow progress for each contact in sequences. Each flow/step gets its own record.';
COMMENT ON COLUMN sequence_contacts.sequence_stepid IS 'References the specific step being processed';
COMMENT ON COLUMN sequence_contacts.processing_device_id IS 'Device currently processing this flow';
COMMENT ON COLUMN sequence_contacts.retry_count IS 'Number of failed attempts for this flow';
COMMENT ON COLUMN device_load_balance.messages_hour IS 'Messages sent in current hour (auto-resets)';
COMMENT ON COLUMN device_load_balance.messages_today IS 'Messages sent today (auto-resets at midnight)';

-- 22. Grant necessary permissions (adjust roles as needed)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO your_app_role;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO your_app_role;

-- Migration complete!
-- This schema supports:
-- - Individual flow tracking per contact
-- - 3000 device load balancing
-- - Automatic progress updates
-- - Schedule time respecting
-- - Retry handling
-- - Performance monitoring