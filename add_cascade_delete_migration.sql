-- Migration to add CASCADE DELETE for device deletions
-- This will ensure that when a device is deleted, all related data is also deleted

-- WARNING: Back up your database before running this migration!

-- 1. First, check and remove existing foreign key constraints if they exist
SET @dbname = DATABASE();

-- Check broadcast_messages foreign key
SELECT @constraint_name := CONSTRAINT_NAME 
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = @dbname
AND TABLE_NAME = 'broadcast_messages'
AND COLUMN_NAME = 'device_id'
AND REFERENCED_TABLE_NAME = 'user_devices';

SET @sql = IF(@constraint_name IS NOT NULL,
    CONCAT('ALTER TABLE broadcast_messages DROP FOREIGN KEY ', @constraint_name),
    'SELECT "No foreign key to drop for broadcast_messages"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2. Add foreign key with CASCADE DELETE for broadcast_messages
ALTER TABLE broadcast_messages
ADD CONSTRAINT fk_broadcast_device
FOREIGN KEY (device_id) 
REFERENCES user_devices(id)
ON DELETE CASCADE
ON UPDATE CASCADE;

-- 3. Check if leads table has device_id column and add CASCADE DELETE
SELECT COUNT(*) INTO @has_device_column
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = @dbname
AND TABLE_NAME = 'leads' 
AND COLUMN_NAME = 'device_id';

-- If leads has device_id, add the constraint
SET @sql = IF(@has_device_column > 0,
    'ALTER TABLE leads 
     ADD CONSTRAINT fk_leads_device 
     FOREIGN KEY (device_id) 
     REFERENCES user_devices(id) 
     ON DELETE SET NULL 
     ON UPDATE CASCADE',
    'SELECT "Leads table does not have device_id column"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 4. Check for other tables that might reference user_devices
-- Add CASCADE DELETE for sequence_contacts if it has device_id
SELECT COUNT(*) INTO @has_seq_device
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = @dbname
AND TABLE_NAME = 'sequence_contacts' 
AND COLUMN_NAME = 'device_id';

SET @sql = IF(@has_seq_device > 0,
    'ALTER TABLE sequence_contacts 
     ADD CONSTRAINT fk_seq_contacts_device 
     FOREIGN KEY (device_id) 
     REFERENCES user_devices(id) 
     ON DELETE CASCADE 
     ON UPDATE CASCADE',
    'SELECT "sequence_contacts table does not have device_id column"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 5. Show summary of changes
SELECT 
    'Foreign Key Constraints Added' as Status,
    'Now when a device is deleted, all related broadcast messages will be automatically deleted' as Description
UNION ALL
SELECT 
    'Leads Protection',
    'If a device is deleted, leads will have their device_id set to NULL (not deleted)'
UNION ALL
SELECT 
    'Data Integrity',
    'No more orphaned messages from deleted devices!';

-- 6. Verify the constraints were added
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    CONSTRAINT_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = @dbname
AND REFERENCED_TABLE_NAME = 'user_devices'
ORDER BY TABLE_NAME;
