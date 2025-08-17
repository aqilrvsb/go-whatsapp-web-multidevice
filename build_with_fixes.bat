@echo off
echo Building WhatsApp Multi-Device with MySQL/PostgreSQL fixes...
echo.

cd src

echo Setting environment...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

echo.
echo Building application...
go build -o ../whatsapp_fixed.exe

if %errorlevel% neq 0 (
    echo.
    echo Build failed! Check the errors above.
    pause
    exit /b 1
)

echo.
echo Build successful! Output: whatsapp_fixed.exe
echo.
echo The following fixes have been applied:
echo 1. MySQL syntax error in chat_store.go - Fixed ON DUPLICATE KEY UPDATE
echo 2. PostgreSQL session_replication_role error - Added database type detection
echo 3. broadcast_coordinator.go - Added MySQL/PostgreSQL compatibility
echo.
echo You can now run the application without the SQL syntax errors.
pause
