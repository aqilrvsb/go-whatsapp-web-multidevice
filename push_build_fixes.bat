@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Committing and pushing current fixes...
git add -A
git commit -m "Fix: Resolved duplicate functions and build errors in app.go

- Removed duplicate GetSequenceSummary, GetWorkerStatus, min, max, countConnectedDevices functions
- Fixed unused device variable in loops
- Added missing StopAllWorkers function
- Fixed corrupted file ending
- Changed device.Name to device.DeviceName"

git push origin main --force

echo.
echo ==========================================
echo Pushed to GitHub!
echo ==========================================
echo.
echo The build should now work properly with all
echo duplicate functions removed and syntax fixed.
echo.
pause
