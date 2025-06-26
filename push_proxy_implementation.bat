@echo off
echo Implementing Malaysian Proxy Support...

cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo.
echo Running go mod tidy...
cd src
go mod tidy
cd ..

echo.
echo Git status...
git status

echo.
echo Adding all changes...
git add .

echo.
echo Committing changes...
git commit -m "feat: Add Malaysian proxy support for ban prevention

- Implement automatic proxy fetching from multiple sources
- Add proxy manager with device assignment
- Create proxied WhatsApp client wrapper
- Add REST API endpoints for proxy management
- Update configuration for proxy settings
- Auto-fetch free Malaysian proxies
- Each device gets unique proxy IP
- Automatic proxy rotation and failover
- Zero configuration required"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo ✅ Successfully pushed Malaysian proxy implementation to GitHub!
pause