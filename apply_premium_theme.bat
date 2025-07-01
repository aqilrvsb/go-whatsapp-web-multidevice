@echo off
echo ========================================
echo Applying Premium Purple/Pink Theme
echo ========================================
echo.

cd /d C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main\src\views

echo Creating backup of dashboard.html...
copy "dashboard.html" "dashboard_backup_theme_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%.html" >nul
echo Backup created!

echo.
echo Applying premium theme changes...

REM PowerShell script to update button classes and other elements
powershell -Command "& {
    $file = 'dashboard.html'
    $content = Get-Content $file -Raw

    # Update all primary buttons to gradient buttons
    $content = $content -replace 'class=""btn btn-primary""', 'class=""btn btn-gradient""'
    $content = $content -replace 'btn btn-primary', 'btn btn-gradient'
    
    # Update success buttons
    $content = $content -replace 'btn btn-success', 'btn btn-gradient'
    
    # Update metric cards
    $content = $content -replace 'class=""metric-card""', 'class=""metric-card glass-card hover-scale""'
    
    # Update navbar
    $content = $content -replace 'navbar navbar-expand-lg navbar-light bg-white', 'navbar navbar-expand-lg'
    
    # Update cards
    $content = $content -replace 'class=""card""', 'class=""card glass-card""'
    $content = $content -replace 'class=""card ', 'class=""card glass-card '
    
    # Update device cards
    $content = $content -replace 'device-card', 'device-card glass-card hover-scale'
    
    # Update tables
    $content = $content -replace 'class=""table', 'class=""table table-dark'
    
    # Update modals
    $content = $content -replace 'modal-content', 'modal-content glass-card'
    
    # Update tab pills
    $content = $content -replace 'nav nav-tabs', 'nav nav-tabs premium-tabs'
    
    # Remove old inline styles that conflict
    $content = $content -replace ':root \{[\s\S]*?\}', ''
    
    # Save the updated content
    Set-Content $file $content -NoNewline -Encoding UTF8
    Write-Host 'Premium theme applied successfully!' -ForegroundColor Green
}"

echo.
echo ========================================
echo Premium Theme Applied!
echo ========================================
echo.
echo Changes made:
echo - All buttons now use gradient style
echo - Cards have glass morphism effect
echo - Premium hover animations added
echo - Purple/Pink color scheme applied
echo - Modern dark theme with gradients
echo.
echo Your dashboard now has a stunning premium look!
echo.
pause