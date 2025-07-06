@echo off
echo ========================================
echo Fixing Trigger Issues
echo ========================================

cd src

echo.
echo Step 1: Backing up files...
copy views\dashboard.html views\dashboard_backup_trigger.html
copy views\device_leads.html views\device_leads_backup_trigger.html

echo.
echo Step 2: Applying fixes...
echo Files will be updated to:
echo - Show trigger column in leads
echo - Replace Start/End triggers with single Trigger field
echo - Update labels in sequences

pause
