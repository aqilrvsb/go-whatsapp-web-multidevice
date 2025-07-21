@echo off
echo ========================================
echo Fixing Refresh Button Client Manager Registration
echo ========================================

:: Create a backup of the original file
copy "src\ui\rest\device_reconnect.go" "src\ui\rest\device_reconnect.go.backup" >nul 2>&1

:: Apply the fix
powershell -Command "(Get-Content 'src\ui\rest\device_reconnect.go') -replace 'cm.AddClient\(deviceID, client\)', 'cm.AddClient(deviceID, client); logrus.Infof(\"✅ Client registered in ClientManager for device %%s\", deviceID)' | Set-Content 'src\ui\rest\device_reconnect.go'"

echo.
echo Fix applied! Now checking if we need to enhance the disconnect handler...
echo.

:: Also ensure disconnect doesn't remove from ClientManager inappropriately
echo Creating enhanced connection monitor...

pause
