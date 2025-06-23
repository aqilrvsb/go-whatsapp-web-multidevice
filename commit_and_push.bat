@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Fix build error: remove unused imports in cmd/rest.go"
git push origin main
echo Done!