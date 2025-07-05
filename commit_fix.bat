@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git commit -m "Fix auto-reconnect database column name issue - use device_name instead of name"
git push origin main
echo Done!
pause