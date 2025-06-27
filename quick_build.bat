@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src
echo Building with campaign fixes...

set CGO_ENABLED=0
go build -o ..\whatsapp.exe main.go

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!
pause
