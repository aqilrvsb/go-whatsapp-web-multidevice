@echo off
echo Committing and pushing changes...

cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

git add -A

git commit -m "Add comprehensive logging to UltraOptimizedBroadcastProcessor to debug campaign message processing"

git push origin main

echo Done!
pause
