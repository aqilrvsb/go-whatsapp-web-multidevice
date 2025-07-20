@echo off
echo "Fixing to use recipient_name from broadcast_messages..."

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
git commit -m "Use recipient_name from broadcast_messages for consistency

- Both campaigns and sequences now use recipient_name from broadcast_messages
- This provides one consistent source for names
- Added fallback to phone if name is empty
- Added logging to track what name is being used
- This ensures the name shown in greetings matches what's in broadcast_messages"

echo "Pushing to GitHub..."
git push origin main

echo "Done! Now using recipient_name from broadcast_messages for all greetings."
