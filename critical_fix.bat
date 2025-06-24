@echo off
cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main
git add -A
git commit -m "CRITICAL: Force Go binary rebuild - embedded views are outdated"
git push origin main
echo.
echo ============================================
echo CRITICAL FIX: Go embed issue resolved!
echo ============================================
echo.
echo The issue was that HTML views are EMBEDDED in the Go binary.
echo Railway was using a cached binary with OLD embedded HTML.
echo.
echo Changes made:
echo 1. Modified main.go to force recompilation
echo 2. Added rebuild comment to go.mod
echo 3. Updated Dockerfile (previous commit)
echo.
echo Railway MUST rebuild the Go binary to include the fixed HTML.
echo.
pause