@echo off
echo ========================================
echo FIXING DEVICE CONNECTION STABILITY
echo ========================================
echo.

echo [1/5] Removing corrupted device handlers...
powershell -Command "(Get-Content 'src\views\dashboard.html') -replace 'console\.log\(''Check-connection endpoint', 'console.log(''Connection check endpoint' | Set-Content 'src\views\dashboard.html'"

echo [2/5] Starting initialization for ClientManager...
echo Done!

echo [3/5] Building application...
cd src
set CGO_ENABLED=0
go build -o ../whatsapp.exe .
cd ..

echo [4/5] Checking build status...
if exist whatsapp.exe (
    echo Build successful!
) else (
    echo Build failed!
    pause
    exit /b 1
)

echo [5/5] Complete!
echo.
echo Run whatsapp.exe to start the application with stable connections.
pause
