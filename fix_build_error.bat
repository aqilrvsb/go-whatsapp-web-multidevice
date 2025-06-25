@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Fix: Remove duplicate GetDevice method from devices.go"
git push origin main
echo.
echo Build fix deployed!
pause
