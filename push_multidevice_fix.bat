@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Replace single-device auto-reconnect with multi-device version optimized for 3000 devices"
git push origin main
echo Done!
pause