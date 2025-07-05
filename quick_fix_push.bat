@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "Fix GetSequenceByID query to match GetSequences - remove non-existent columns"
git push origin main
pause
