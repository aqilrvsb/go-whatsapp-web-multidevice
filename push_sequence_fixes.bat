@echo off
echo Pushing sequence fixes to GitHub...

REM Add all changes
git add -A

REM Commit with descriptive message
git commit -m "Fix sequence creation: niche, time_schedule, and steps saving

- Added niche field saving in sequence creation
- Added time_schedule field saving and display
- Fixed sequence steps not being saved properly
- Added min/max delay fields to sequence creation
- Updated frontend to show niche and schedule time
- Fixed step creation with all required fields (day_number, message_type, etc.)
- Enhanced sequence response to include all fields"

REM Push to main branch
git push origin main

echo Push complete!
pause
