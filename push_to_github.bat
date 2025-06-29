@echo off
echo ========================================
echo Pushing to GitHub main branch
echo ========================================

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Current directory:
cd

echo.
echo Adding all changes...
git add -A

echo.
echo Creating commit...
git commit -m "Fix: Worker health check and auto-reconnect system - All control buttons functional

- Fixed syntax errors in rest.go (backtick issues)
- Added GetAllDevices method to UserRepository
- Fixed duplicate method declarations in device_worker.go
- Implemented device health monitor with auto-reconnect
- Enhanced client manager with better registration
- Improved worker health checks
- All worker control buttons now functional
- Fixed compilation errors and unused imports
- Better error handling and recovery mechanisms
- Updated README with comprehensive documentation"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo ========================================
echo Push Complete!
echo ========================================
echo.
echo Changes have been pushed to GitHub main branch.
echo Railway should auto-deploy these changes.
echo.
pause
