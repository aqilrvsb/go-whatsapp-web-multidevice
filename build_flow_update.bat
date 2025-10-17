@echo off
echo === Flow Update Feature - Build and Deploy ===
echo.

cd /d C:\Users\aqilz\go-whatsapp-web-multidevice-main

echo Setting CGO_ENABLED=0...
set CGO_ENABLED=0

echo.
echo Building without CGO...
cd src
go build -o ..\whatsapp.exe

if %errorlevel% neq 0 (
    echo.
    echo ❌ Build failed! Checking for errors...
    pause
    exit /b 1
)

cd ..

echo.
echo ✅ Build successful!
echo Executable: whatsapp.exe
echo.
echo === Committing and pushing to GitHub ===
git add .
git commit -m "Add Flow Update feature - Continue sequences from last day"
git push origin main

if %errorlevel% neq 0 (
    echo.
    echo ❌ Git push failed!
    pause
    exit /b 1
)

echo.
echo ✅ === Complete ===
echo.
echo 🚀 Railway will auto-deploy from main branch
echo.
pause
