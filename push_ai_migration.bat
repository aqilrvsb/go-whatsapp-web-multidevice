@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding all changes...
git add -A

echo Committing changes...
git commit -m "Add AI Campaign tables to auto-migration system - tables will be created automatically on startup"

echo Pushing to main branch...
git push origin main

echo Done!
echo.
echo NOTE: The AI Campaign tables will be created automatically when the application starts.
echo No manual SQL execution needed!
echo.
pause