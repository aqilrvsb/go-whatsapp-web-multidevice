@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add src/infrastructure/whatsapp/auto_reconnect.go
git commit -m "Fix auto-reconnect use device_name column"
git push origin main
echo Done!
pause