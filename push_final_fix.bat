@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Refactor: Clean multi-device architecture - remove single-device functions, add proper multi-device auto-reconnect"
git push origin main
echo Done!
pause