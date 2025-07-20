@echo off
echo "Fixing registration endpoint authentication..."

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
git commit -m "Fix registration endpoint - add /api/register to public routes

- Registration was returning 401 Unauthorized
- Added /api/register to PublicRoutes in custom_auth.go
- Now registration endpoint is accessible without authentication"

echo "Pushing to GitHub..."
git push origin main

echo "Done! Registration endpoint is now public and should work."
