@echo off
echo ========================================
echo Pushing README update
echo ========================================

cd /d "C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main"

REM Add README
echo Adding README...
git add README.md

REM Commit
echo Committing changes...
git commit -m "Update README: Add auto-reconnect behavior details

- Added new section explaining auto-reconnect behavior
- Clarified that devices are NOT set to offline automatically
- Explained actual reconnection using GetOrCreateDeviceConnection
- Added details about status preservation and graceful failures
- Updated Railway restart behavior description"

REM Push to main branch
echo Pushing to main branch...
git push origin main

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Successfully pushed README update!
) else (
    echo.
    echo ❌ Push failed!
)

pause
