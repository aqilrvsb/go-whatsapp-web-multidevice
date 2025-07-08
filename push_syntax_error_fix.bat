@echo off
echo ========================================
echo Pushing Critical Syntax Error Fix to GitHub
echo ========================================

echo.
echo Changes made:
echo - Fixed syntax error in sequence-summary-tab event listener
echo - Added debug logging to help diagnose data issues
echo - Added fallback value (0) for inactive count display
echo - This should finally fix the undefined display issue
echo.

REM Add all changes
git add -A

REM Commit with descriptive message
git commit -m "fix: Critical syntax error in sequence summary tab handler

- Fixed missing document.getElementById in tab event listener
- Added console.log debugging to track data flow
- Added fallback || 0 for inactive count display
- This was preventing loadSequenceSummary from being called when tab clicked"

REM Push to main branch
git push origin main

echo.
echo ========================================
echo Push completed!
echo ========================================
pause
