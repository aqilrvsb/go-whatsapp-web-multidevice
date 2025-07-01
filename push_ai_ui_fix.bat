@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo Adding all changes...
git add -A

echo Committing changes...
git commit -m "Add AI Leads Management UI - exact copy of device leads with import/export functionality"

echo Pushing to main branch...
git push origin main

echo Done!
echo.
echo IMPORTANT: Run the following SQL in your database to create the AI tables:
echo.
echo 1. Connect to your database
echo 2. Run: create_ai_tables.sql
echo.
pause