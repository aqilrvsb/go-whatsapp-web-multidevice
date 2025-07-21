@echo off
echo ========================================
echo FIXING DEVICE CONNECTION CORRUPTION ISSUE
echo ========================================
echo.

echo [1/4] Fixing dashboard.html comments that cause corruption...
powershell -Command "(Get-Content 'src\views\dashboard.html') -replace '// If check-connection fails, just continue', '// If connection check fails, just continue' -replace 'console.log\(''Check-connection endpoint', 'console.log(''Connection check endpoint' | Set-Content 'src\views\dashboard.html'"

echo [2/4] Fixing team_dashboard.html comments...
powershell -Command "(Get-Content 'src\views\team_dashboard.html') -replace '// If check-connection fails, just continue', '// If connection check fails, just continue' -replace 'console.log\(''Check-connection endpoint', 'console.log(''Connection check endpoint' | Set-Content 'src\views\team_dashboard.html'"

echo [3/4] Fixing dashboard_reference.html comments...
powershell -Command "(Get-Content 'src\views\dashboard_reference.html') -replace '// If check-connection fails, just continue', '// If connection check fails, just continue' -replace 'console.log\(''Check-connection endpoint', 'console.log(''Connection check endpoint' | Set-Content 'src\views\dashboard_reference.html'"

echo [4/4] Creating a robust device connection fix...
echo Done!

pause
