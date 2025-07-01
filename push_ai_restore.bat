@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding all changes...
git add -A

echo Committing changes...
git commit -m "Restore AI Campaign to simple version - no delivery tracking complexity"

echo Pushing to main branch...
git push origin main

echo Done!
pause