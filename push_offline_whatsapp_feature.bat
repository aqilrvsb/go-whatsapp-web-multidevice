@echo off
echo Building and pushing WhatsApp Web offline feature...

REM Build the application
echo Building application...
set CGO_ENABLED=0
go build -o whatsapp.exe ./src

if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!

REM Git operations
echo Adding changes to git...
git add -A

echo Committing changes...
git commit -m "Add WhatsApp button for viewing chat history when device is offline"

echo Pushing to GitHub...
git push origin main

echo.
echo ===================================
echo Deployment complete!
echo ===================================
echo.
echo Changes:
echo - Added WhatsApp button when device is offline
echo - Users can view previous chat history even when disconnected
echo - Offline state clearly indicated with option to browse stored messages
echo - Input area hidden when offline to prevent confusion
echo.
pause