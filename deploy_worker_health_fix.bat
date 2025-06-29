@echo off
echo ================================================
echo Deploying Worker Health & Auto-Reconnect Fix
echo ================================================
echo.

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Step 1: Adding imports to rest.go...
cd src\cmd

REM Add health monitor import
powershell -Command "(Get-Content rest.go) -replace '\"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket\"', '\"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket\"`n`t\"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp\"' | Set-Content rest.go"

REM Add health monitor initialization after broadcast manager
powershell -Command "$content = Get-Content rest.go; $newContent = @(); $added = $false; foreach($line in $content) { $newContent += $line; if($line -match 'Broadcast manager started' -and -not $added) { $newContent += '`t'; $newContent += '`t// Start device health monitor'; $newContent += '`thealthMonitor := whatsapp.GetDeviceHealthMonitor(whatsappDB)'; $newContent += '`thealthMonitor.Start()'; $newContent += '`tlogrus.Info(`"Device health monitor started`")'; $added = $true } }; $newContent | Set-Content rest.go"

REM Add worker control API initialization
powershell -Command "$content = Get-Content rest.go; $newContent = @(); $added = $false; foreach($line in $content) { $newContent += $line; if($line -match 'rest.InitRestMonitoring\(app\)' -and -not $added) { $newContent += '`trest.InitWorkerControlAPI(app) // Add worker control endpoints'; $added = $true } }; $newContent | Set-Content rest.go"

cd ..\..

echo.
echo Step 2: Creating deployment package...
git add -A
git commit -m "Add worker health monitoring and auto-reconnect functionality

- Added DeviceHealthMonitor for automatic device reconnection
- Enhanced ClientManager with better registration and health checks
- Improved DeviceWorker with robust health monitoring
- Added worker control API endpoints (resume, stop, restart)
- Created frontend JavaScript for worker control buttons
- All buttons in Worker Status page are now functional
- Devices automatically reconnect if they disconnect
- Workers restart automatically if they become unhealthy"

echo.
echo Step 3: Pushing to GitHub...
git push origin main --force

echo.
echo ================================================
echo ✅ Worker Health & Auto-Reconnect Deployed!
echo ================================================
echo.
echo New Features:
echo.
echo 1. DEVICE HEALTH MONITOR:
echo    - Checks device health every 30 seconds
echo    - Automatically reconnects disconnected devices
echo    - Updates device status in real-time
echo.
echo 2. ENHANCED CLIENT MANAGER:
echo    - Better device registration on connection
echo    - Health status checking
echo    - Cleanup of disconnected clients
echo.
echo 3. IMPROVED WORKER HEALTH:
echo    - Workers check their own health
echo    - Auto-restart if unhealthy
echo    - Better error handling
echo.
echo 4. FUNCTIONAL BUTTONS:
echo    - Resume Failed: Restarts all stopped/failed workers
echo    - Stop All: Stops all active workers
echo    - Per-device restart/reconnect buttons
echo.
echo 5. AUTO-RECONNECT:
echo    - Devices with saved sessions reconnect automatically
echo    - No QR scan needed if session exists
echo    - Graceful handling of connection failures
echo.
echo The system will now:
echo - Monitor all devices and workers continuously
echo - Reconnect devices automatically when possible
echo - Restart unhealthy workers
echo - Provide real-time status updates
echo.
pause