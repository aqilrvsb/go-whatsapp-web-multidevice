@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Fix auto-reconnect: increase delay to 30s and use correct device_name column"
git push origin main
echo Done!
pause