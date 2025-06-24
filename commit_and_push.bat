@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Replace bcrypt with base64 encoding for passwords - simpler personal system"
git push origin main
echo Done!