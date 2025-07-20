# Greeting Fix Summary - January 2025

## Issues Fixed ✅

### 1. Customer Name Not Showing ✅
**Problem**: Greeting showed "Hi Cik" instead of using the actual customer name

**Root Cause**: 
- For sequences: The `contact_name` field in `sequence_contacts` table might contain phone numbers or be empty
- The greeting processor wasn't properly detecting when names were phone numbers

**Fix Applied**:
- Enhanced name detection logic in `greeting_processor.go`
- Added check for names that match phone numbers (even with formatting differences)
- Added debug logging to track names throughout the flow
- Properly falls back to "Cik" when name is empty, whitespace only, or a phone number

### 2. Line Break Issue ✅ FIXED!
**Problem**: Messages showed "Hi Cik, apa khabarMasa awal..." without line breaks

**Root Cause**: 
- The `MessageRandomizer` was using `strings.Fields()` which splits text by ALL whitespace
- This was converting `\n\n` (double newline) into a single space

**Fix Applied**:
- Rewrote `insertZeroWidthSpaces()` to preserve newlines
- Now carefully inserts zero-width characters without destroying formatting
- Line breaks are preserved throughout the entire message flow

**Result**: Messages now display properly as:
```
Hi [Name],

[Your message content]
```

### 3. Debug Logging Added
To help troubleshoot future issues, we added logging at key points:
- `[ENROLLMENT]` - Shows what name is stored when lead enrolls
- `[SEQUENCE-NAME]` - Shows what name is retrieved from database  
- `[GREETING]` - Shows greeting generation details

## How Names Work

### For Sequences:
1. Lead enrolls → `lead.name` is copied to `sequence_contacts.contact_name`
2. When processing → Uses `contact_name` from `sequence_contacts` table
3. If name is empty/phone → Falls back to "Cik"

### For Campaigns:
1. Uses `name` directly from `leads` table
2. Same fallback logic applies

## Testing the Fix

1. Check Railway logs for the debug messages:
   ```
   [ENROLLMENT] Created step 1 for 60123456789 - Name: 'John Doe', status: pending
   [SEQUENCE-NAME] Contact: 60123456789, Name from sequence_contacts: 'John Doe'
   [GREETING] Name: 'John Doe', Phone: 60123456789, Greeting: 'Hi John Doe,'
   ```

2. If name is still showing as "Cik", check:
   - What's stored in the `leads.name` field
   - What's stored in `sequence_contacts.contact_name` field
   - The debug logs to see where the name gets lost

## Database Check Queries

```sql
-- Check lead names
SELECT phone, name FROM leads WHERE phone = '60123456789';

-- Check sequence contact names
SELECT contact_phone, contact_name 
FROM sequence_contacts 
WHERE contact_phone = '60123456789';

-- Update name if needed
UPDATE leads SET name = 'Customer Name' WHERE phone = '60123456789';
```

## Next Steps

If issues persist:
1. Check the debug logs to see what names are being used
2. Verify the database has proper names (not phone numbers)
3. For platform APIs (Wablas/Whacenter), may need special encoding for line breaks
