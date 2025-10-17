-- Fix NULL phone values in user_devices table
-- Run this SQL to fix your sequence processing issue

-- 1. Check how many devices have NULL phone values
SELECT COUNT(*) as null_phone_count 
FROM user_devices 
WHERE phone IS NULL;

-- 2. Show which devices have NULL phone (to review before fixing)
SELECT id, user_id, device_name, phone, status, jid, created_at
FROM user_devices 
WHERE phone IS NULL;

-- 3. Fix the NULL phone values - Choose one of these options:

-- Option A: Set to empty string (simplest)
UPDATE user_devices 
SET phone = '' 
WHERE phone IS NULL;

-- Option B: Extract phone from JID if available
UPDATE user_devices 
SET phone = SUBSTRING_INDEX(jid, '@', 1)
WHERE phone IS NULL AND jid IS NOT NULL AND jid != '';

-- Option C: Set a placeholder value
UPDATE user_devices 
SET phone = 'NOT_SET'
WHERE phone IS NULL;

-- 4. Prevent future NULLs by modifying the column
ALTER TABLE user_devices 
MODIFY COLUMN phone VARCHAR(255) NOT NULL DEFAULT '';

-- 5. Verify the fix
SELECT COUNT(*) as remaining_null_phones
FROM user_devices 
WHERE phone IS NULL;

-- Should return 0
