@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding all changes...
git add -A

echo Committing changes...
git commit -m "Fix AI Campaign: Sequential processing, no lead failed status, device ban handling"

echo Pushing to main branch...
git push origin main

echo Done!
pause