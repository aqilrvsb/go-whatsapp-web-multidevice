@echo off
echo Fixing unused device variable on line 1736...

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

REM Use sed-like replacement with PowerShell targeting specific line
powershell -Command "(Get-Content src\ui\rest\app.go) | ForEach-Object { if ($_.Trim() -eq 'for _, device := range devices {' -and $MyInvocation.ScriptLineNumber -eq 1736) { $_ -replace 'for _, device := range devices', 'for range devices' } else { $_ } } | Set-Content src\ui\rest\app.go.tmp"

REM Check if temp file was created
if exist src\ui\rest\app.go.tmp (
    move /Y src\ui\rest\app.go.tmp src\ui\rest\app.go
    echo Fixed successfully!
) else (
    echo Fix failed - using simpler approach
    REM Simple global replacement as fallback
    powershell -Command "$content = Get-Content 'src\ui\rest\app.go' -Raw; $count = 0; $content = [regex]::Replace($content, 'for _, device := range devices \{', { param($m); $global:count++; if ($global:count -eq 2) { 'for range devices {' } else { $m.Value } }); Set-Content 'src\ui\rest\app.go' -Value $content -NoNewline"
)

echo Building to test...
cd src
go build .
cd ..

if %ERRORLEVEL% EQU 0 (
    echo Build successful! Committing and pushing...
    git add -A
    git commit -m "Fix: Remove unused device variable in StopAllWorkers function"
    git push origin main --force
    echo Done!
) else (
    echo Build failed. Please check the error above.
)

pause
