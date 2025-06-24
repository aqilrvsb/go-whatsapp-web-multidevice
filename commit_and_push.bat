@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Fix login issue: properly create admin user with bcrypt hash"
git push origin main
echo Done!