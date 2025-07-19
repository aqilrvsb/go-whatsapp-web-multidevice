@echo off
echo === Committing and Pushing All Fixes ===

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo.
echo Adding all changes...
git add -A

echo.
echo Creating commit...
git commit -m "Fix sequence contacts and platform device detection

FIXED ISSUES:
1. Sequence Contacts ON CONFLICT:
   - Added proper unique constraint (sequence_id, contact_phone, sequence_stepid)
   - Cleaned up orphaned records without Step 1
   - Database now ready for proper sequence enrollment

2. Platform Device Detection:
   - Fixed syntax errors in platform checks (was: Platform != "", now: if device.Platform != "")
   - Platform devices (Wablas/Whacenter) now properly route to APIs
   - Fixed auto_reconnect.go compilation error

3. Database Cleanup:
   - Removed completed sequence contacts
   - Added constraint to prevent duplicate enrollments
   - Ready for fresh testing

This ensures:
- Sequences enroll properly without ON CONFLICT errors
- Platform devices use external APIs instead of WhatsApp Web
- Steps activate by earliest trigger time
- No missing or duplicate steps"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo === Push Complete! ===
pause