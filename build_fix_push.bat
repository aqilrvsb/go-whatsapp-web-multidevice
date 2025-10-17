@echo off
echo ========================================
echo Building without CGO for Linux...
echo ========================================

cd /d %~dp0

echo [1/4] Setting environment variables...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

echo [2/4] Building application...
cd src
go build -ldflags="-s -w" -o ../whatsapp main.go
if %errorlevel% neq 0 (
    echo [ERROR] Build failed!
    pause
    exit /b 1
)

echo [SUCCESS] Build successful!
cd ..

echo.
echo [3/4] Committing changes...
git add .
git commit -m "Fix: Remove device status checks for Whacenter - Build without CGO"

echo.
echo [4/4] Pushing to GitHub...
git push -f origin master:main

echo.
echo ========================================
echo DONE! Railway will auto-deploy now.
echo ========================================
pause
