@echo off
cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add src/ui/rest/app.go
git commit -m "Fix syntax error in GetCampaigns function"
git push origin main
echo Syntax fix pushed successfully!
pause