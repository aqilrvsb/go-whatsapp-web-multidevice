@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Add debug logging to check user_devices columns and use device_name"
git push origin main
echo Done!
pause