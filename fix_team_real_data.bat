@echo off
echo ========================================
echo Fixing Team Dashboard Real Data Display
echo ========================================

cd /d "C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src"

echo Building with CGO_ENABLED=0...
set CGO_ENABLED=0
go build -o ..\whatsapp.exe .

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!
echo.
echo Committing changes...
cd ..
git add .
git commit -m "Fix team dashboard to show real data filtered by assigned devices"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo ========================================
echo Fix deployed! Railway will auto-deploy.
echo ========================================
pause
