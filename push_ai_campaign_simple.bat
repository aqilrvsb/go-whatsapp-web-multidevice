@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding all changes...
git add -A

echo Committing changes...
git commit -m "AI Campaign: Keep simple approach - no delivery tracking, assume success if device online"

echo Pushing to main branch...
git push origin main

echo Done!
pause