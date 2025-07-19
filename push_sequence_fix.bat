@echo off
echo === Committing and Pushing Sequence Fix ===

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo.
echo Adding files to git...
git add -A

echo.
echo Creating commit...
git commit -m "Fix sequence contacts for 3000 devices - activate by earliest trigger time

- Fixed updateContactProgress to activate by earliest trigger_time, not step number
- Added FOR UPDATE SKIP LOCKED for concurrent access by 3000 devices  
- Added unique constraint to prevent duplicate active steps
- Added transaction isolation level for proper concurrency
- Fixed race conditions that caused missing Step 1 and duplicate Step 2
- Optimized indexes for fast pending step lookup
- Database function for atomic step progression

This ensures proper sequence flow: pending -> active -> completed
Each contact can only have ONE active step at a time
Steps are activated in order of their scheduled time, not step number"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo === Push Complete! ===
echo.
echo The fix includes:
echo - Database optimizations for 3000 concurrent devices
echo - Proper step activation by earliest trigger time
echo - Prevention of duplicate active records
echo - Race condition fixes
echo.
pause