@echo off
echo ========================================
echo Pushing authentication fix
echo ========================================

cd /d "C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main"

REM Add changed files
echo Adding changes...
git add src/ui/rest/device_multidevice.go
git add src/ui/rest/middleware/custom_auth.go

REM Commit
echo Committing changes...
git commit -m "Fix: Authentication for device refresh endpoint

- DeviceConnect now checks session cookie if userID not in context
- Removed /api and /app from public routes in middleware
- Added fallback session validation for API endpoints
- Fixes 401 Unauthorized error when clicking refresh button"

REM Push to main branch
echo Pushing to main branch...
git push origin main

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Successfully pushed authentication fix!
    echo.
    echo The refresh button should now work properly.
) else (
    echo.
    echo ❌ Push failed!
)

pause
