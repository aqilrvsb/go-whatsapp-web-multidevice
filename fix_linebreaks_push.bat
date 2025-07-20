@echo off
echo "Fixing line break issue in message randomizer..."

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
git commit -m "CRITICAL FIX: Line breaks now preserved in messages

- Fixed message randomizer destroying double newlines
- insertZeroWidthSpaces was using strings.Fields which removes line breaks
- Now preserves all formatting including \n\n for proper WhatsApp display
- Added better logging to show escaped characters for debugging
- Messages should now show proper line breaks between greeting and content"

echo "Pushing to GitHub..."
git push origin main

echo "Done! Line break fix pushed to GitHub."
