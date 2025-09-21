@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Add duplicate prevention for campaigns using SQL NOT EXISTS check"
git push origin main
echo Done!