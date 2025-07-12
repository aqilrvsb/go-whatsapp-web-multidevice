@echo off
echo ========================================
echo Fixing Team Dashboard - Device Assignment & UI
echo ========================================

cd /d "C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src"

echo Building with CGO_ENABLED=0...
set CGO_ENABLED=0
go build -o ..\whatsapp.exe .

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo Build successful!
echo.
echo Committing changes...
cd ..
git add .
git commit -m "Fix team dashboard - add proper device assignment system and match admin UI"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo ========================================
echo Fix deployed! Railway will auto-deploy.
echo.
echo IMPORTANT: After deployment, run this SQL in your database:
echo.
echo -- Create team member device assignments table
echo CREATE TABLE IF NOT EXISTS team_member_devices (
echo     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
echo     team_member_id UUID NOT NULL REFERENCES team_members(id) ON DELETE CASCADE,
echo     device_id UUID NOT NULL REFERENCES user_devices(id) ON DELETE CASCADE,
echo     assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
echo     assigned_by UUID REFERENCES users(id),
echo     UNIQUE(team_member_id, device_id)
echo );
echo.
echo CREATE INDEX idx_team_member_devices_member ON team_member_devices(team_member_id);
echo CREATE INDEX idx_team_member_devices_device ON team_member_devices(device_id);
echo.
echo Then assign devices to team members via the API or database.
echo ========================================
pause
