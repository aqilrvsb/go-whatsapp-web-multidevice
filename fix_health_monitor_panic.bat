@echo off
echo ========================================
echo Fixing Device Health Monitor Nil Pointer
echo ========================================

cd src

:: Backup the original file
copy "infrastructure\whatsapp\device_health_monitor.go" "infrastructure\whatsapp\device_health_monitor.go.backup" >nul 2>&1

:: Fix the nil pointer issue
powershell -Command "$content = Get-Content 'infrastructure\whatsapp\device_health_monitor.go' -Raw; $content = $content -replace '// Check if client is connected\r?\n\s*if !client\.IsConnected\(\)', '// Check if client exists and is connected`r`n`tif client == nil {`r`n`t`tlogrus.Warnf(\"Device %%s has nil client\", deviceID)`r`n`t`tuserRepo.UpdateDeviceStatus(deviceID, \"offline\", device.Phone, device.JID)`r`n`t`treturn`r`n`t} else if !client.IsConnected()'; Set-Content 'infrastructure\whatsapp\device_health_monitor.go' $content"

echo.
echo Building application without CGO...
cd ..
set CGO_ENABLED=0
go build -o whatsapp.exe ./src

if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo ========================================
echo Build successful! 
echo ========================================
echo.
echo Committing and pushing to GitHub...

git add -A
git commit -m "Fix device health monitor nil pointer panic - Check if client is nil before calling methods"
git push origin main

echo.
echo ========================================
echo Fix deployed successfully!
echo ========================================
pause
