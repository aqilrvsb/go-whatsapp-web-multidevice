-- Fix for NULL phone values in user_devices table
-- This script updates NULL phone values to prevent sequence processing errors

-- First, let's see how many devices have NULL phone values
SELECT COUNT(*) as null_phone_count 
FROM user_devices 
WHERE phone IS NULL;

-- Show the devices with NULL phone values
SELECT id, user_id, device_name, phone, status, jid
FROM user_devices 
WHERE phone IS NULL;

-- Fix 1: Update NULL phone values to empty string
UPDATE user_devices 
SET phone = '' 
WHERE phone IS NULL;

-- Fix 2: Alternatively, if you want to set phone based on JID
-- UPDATE user_devices 
-- SET phone = SUBSTRING_INDEX(jid, '@', 1)
-- WHERE phone IS NULL AND jid IS NOT NULL AND jid != '';

-- Verify the fix
SELECT COUNT(*) as remaining_null_phones
FROM user_devices 
WHERE phone IS NULL;

-- Optional: Add a default value to prevent future NULLs
ALTER TABLE user_devices 
MODIFY COLUMN phone VARCHAR(255) NOT NULL DEFAULT '';
