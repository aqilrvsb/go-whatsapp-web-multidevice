@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
echo ================================================
echo Git Commit History (Last 30 commits)
echo ================================================
echo.
git log --oneline -30
echo.
echo ================================================
echo To restore to a specific version, note the commit hash
echo ================================================
pause