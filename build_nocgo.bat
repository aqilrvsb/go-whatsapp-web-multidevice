@echo off
echo Building WhatsApp Multi-Device without CGO for Linux...

cd /d %~dp0

echo [1/3] Setting environment variables...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

echo [2/3] Building application...
cd src
go build -ldflags="-s -w" -o ../whatsapp main.go

if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Build failed!
    pause
    exit /b 1
)

echo [3/3] Build successful!
echo.
echo Executable created: whatsapp (Linux binary)
echo.
echo Ready to push to GitHub for Railway deployment
echo.
pause
