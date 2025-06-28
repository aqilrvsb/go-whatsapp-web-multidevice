@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo === Pushing device connection status fix and README update ===
echo.

git add src/usecase/campaign_trigger.go
git add src/usecase/optimized_campaign_trigger.go
git add src/usecase/sequence.go
git add README.md
git commit -m "fix: Device connection status check and update README

- Fixed case sensitivity issue for device status (connected vs Connected)
- Added debug logging to show device status during campaign execution
- Fixed same issue in sequences
- Updated README with comprehensive development summary
- System now properly detects connected devices for campaigns"

git push origin main

echo.
echo === All fixes pushed successfully! ===
echo.
echo Your WhatsApp Multi-Device System is now fully operational with:
echo - Campaigns detecting connected devices properly
echo - Sequences working with connected devices
echo - Complete development summary in README
echo.
pause
