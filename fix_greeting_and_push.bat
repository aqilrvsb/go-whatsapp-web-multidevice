@echo off
echo "Fixing greeting issues and pushing to GitHub..."

REM Build the application
echo "Building application..."
SET CGO_ENABLED=0
SET GOOS=windows
SET GOARCH=amd64
cd src
go build -ldflags="-s -w" -o ../whatsapp.exe main.go
cd ..

if %ERRORLEVEL% NEQ 0 (
    echo "Build failed!"
    exit /b 1
)

echo "Build successful!"

REM Git operations
echo "Committing changes..."
git add -A
git commit -m "Fix greeting issues: proper name handling and debug logging

- Added debug logging to track contact names in sequences
- Improved name detection to properly use 'Cik' for empty/phone names
- Added logging in greeting processor for better debugging
- Fixed name comparison logic to handle phone numbers better"

echo "Pushing to GitHub..."
git push origin main

echo "Done! Changes pushed to GitHub."
