-- =====================================================
-- FIX SEQUENCE CONTACTS ISSUES
-- =====================================================
-- This script will:
-- 1. Clean up duplicate and inconsistent data
-- 2. Add proper constraints to prevent future issues
-- 3. Fix the updateContactProgress logic

BEGIN;

-- Step 1: Backup current data
CREATE TABLE IF NOT EXISTS sequence_contacts_backup_fix AS 
SELECT * FROM sequence_contacts;

-- Step 2: Clean up current mess
-- Remove all but keep a record for analysis
DELETE FROM sequence_contacts WHERE status = 'completed';

-- Step 3: Fix duplicate active records (keep only the earliest step)
WITH duplicates AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY sequence_id, contact_phone 
                             ORDER BY current_step ASC) as rn
    FROM sequence_contacts
    WHERE status = 'active'
)
DELETE FROM sequence_contacts
WHERE id IN (SELECT id FROM duplicates WHERE rn > 1);

-- Step 4: Fix steps that should be in order
WITH ordered_steps AS (
    SELECT id, 
           sequence_id,
           contact_phone,
           current_step,
           status,
           LAG(current_step) OVER (PARTITION BY sequence_id, contact_phone ORDER BY current_step) as prev_step
    FROM sequence_contacts
)
UPDATE sequence_contacts sc
SET status = 'pending'
FROM ordered_steps os
WHERE sc.id = os.id
  AND os.status = 'active'
  AND os.prev_step IS NOT NULL
  AND os.current_step > os.prev_step + 1;

-- Step 5: Add missing constraints to prevent issues
-- Drop existing constraint if any
ALTER TABLE sequence_contacts 
DROP CONSTRAINT IF EXISTS check_one_active_per_contact;

-- Add constraint to ensure only one active step per contact
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_active_per_contact
ON sequence_contacts(sequence_id, contact_phone)
WHERE status = 'active';

-- Add constraint to ensure proper step progression
ALTER TABLE sequence_contacts
ADD CONSTRAINT check_valid_status 
CHECK (status IN ('pending', 'active', 'sent', 'failed', 'completed'));

-- Step 6: Add trigger to prevent activation of non-sequential steps
CREATE OR REPLACE FUNCTION check_step_sequence()
RETURNS TRIGGER AS $$
BEGIN
    -- When activating a step, ensure all previous steps are completed
    IF NEW.status = 'active' AND OLD.status = 'pending' THEN
        -- Check if there are any incomplete previous steps
        IF EXISTS (
            SELECT 1 
            FROM sequence_contacts 
            WHERE sequence_id = NEW.sequence_id 
              AND contact_phone = NEW.contact_phone
              AND current_step < NEW.current_step
              AND status NOT IN ('completed', 'sent', 'failed')
        ) THEN
            RAISE EXCEPTION 'Cannot activate step % before completing previous steps', NEW.current_step;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS enforce_step_sequence ON sequence_contacts;
CREATE TRIGGER enforce_step_sequence
BEFORE UPDATE ON sequence_contacts
FOR EACH ROW
EXECUTE FUNCTION check_step_sequence();

-- Step 7: Create a function for safe step progression
CREATE OR REPLACE FUNCTION progress_sequence_contact(
    p_contact_id UUID,
    p_status TEXT DEFAULT 'sent'
) RETURNS VOID AS $$
DECLARE
    v_sequence_id UUID;
    v_contact_phone VARCHAR;
    v_current_step INT;
    v_next_step_id UUID;
BEGIN
    -- Start transaction
    -- Mark current step as completed
    UPDATE sequence_contacts 
    SET status = p_status,
        completed_at = NOW()
    WHERE id = p_contact_id
    RETURNING sequence_id, contact_phone, current_step 
    INTO v_sequence_id, v_contact_phone, v_current_step;
    
    -- Find and activate next step
    SELECT id INTO v_next_step_id
    FROM sequence_contacts
    WHERE sequence_id = v_sequence_id
      AND contact_phone = v_contact_phone
      AND status = 'pending'
      AND current_step = v_current_step + 1
    LIMIT 1;
    
    IF v_next_step_id IS NOT NULL THEN
        UPDATE sequence_contacts
        SET status = 'active'
        WHERE id = v_next_step_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Step 8: View current state after fixes
SELECT 
    contact_phone, 
    current_step, 
    status, 
    current_trigger
FROM sequence_contacts
ORDER BY contact_phone, current_step;

COMMIT;