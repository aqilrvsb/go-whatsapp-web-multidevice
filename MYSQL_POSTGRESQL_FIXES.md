# MySQL 5.7 vs PostgreSQL Compatibility Fixes

## Issues Fixed

### 1. MySQL Syntax Error in chat_store.go
**Problem**: The `ON DUPLICATE KEY UPDATE` query had line breaks that caused MySQL syntax error:
```
Error 1064 (42000): You have an error in your SQL syntax
```

**Fixed**: Removed line breaks in the SQL query in `src/infrastructure/whatsapp/chat_store.go`:
```sql
-- Before (BROKEN):
ON DUPLICATE KEY UPDATE 
    chat_name = VALUES(chat_name),
    last_message_time = VALUES(last_message_time)

-- After (FIXED):
ON DUPLICATE KEY UPDATE chat_name = VALUES(chat_name), last_message_time = VALUES(last_message_time)
```

### 2. PostgreSQL-specific Command in MySQL
**Problem**: Using PostgreSQL's `session_replication_role` in MySQL caused:
```
Error 1193 (HY000): Unknown system variable 'session_replication_role'
```

**Fixed**: Added database type detection in `src/infrastructure/whatsapp/session_cleanup_enhanced.go`:
```go
// MySQL:
SET FOREIGN_KEY_CHECKS = 0/1

// PostgreSQL:
SET session_replication_role = 'replica'/'origin'
```

## Files Modified

1. **src/infrastructure/whatsapp/chat_store.go**
   - Fixed SQL syntax for `ON DUPLICATE KEY UPDATE`
   - Removed problematic line breaks

2. **src/infrastructure/whatsapp/session_cleanup_enhanced.go**
   - Added database type detection
   - Use appropriate foreign key disable commands for each database
   - Added `os` import for environment variable access

## Database Compatibility Guide

### MySQL 5.7+ Syntax:
- Use `ON DUPLICATE KEY UPDATE column = VALUES(column)`
- Use `SET FOREIGN_KEY_CHECKS = 0/1` for FK constraints
- Use `?` for placeholders
- No UUID type (use VARCHAR(36))

### PostgreSQL Syntax:
- Use `ON CONFLICT (columns) DO UPDATE SET column = EXCLUDED.column`
- Use `SET session_replication_role = 'replica'/'origin'` for FK constraints
- Use `$1, $2, $3...` for placeholders
- Native UUID type support

## Environment Detection

The system detects database type by checking:
1. `MYSQL_URI` environment variable
2. `DB_URI` environment variable
3. If contains "postgres" → PostgreSQL, else → MySQL

## Testing

After applying these fixes, the WhatsApp device scanning should work without errors:
- MySQL syntax errors are resolved
- Foreign key operations use correct commands
- History sync completes successfully
