@echo off
echo Pushing import fixes and README updates...
echo.

cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding changes...
git add -A

echo.
echo Committing...
git commit -m "Fix import errors and update README with system assessment

- Removed services import from optimized triggers
- Fixed compilation error by removing IWhatsappService dependency
- Updated README with latest capabilities and architecture
- Added honest system assessment for 3000 devices
- Documented message sending logic clearly
- Added performance specifications"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo Done! All fixes have been pushed.
pause
