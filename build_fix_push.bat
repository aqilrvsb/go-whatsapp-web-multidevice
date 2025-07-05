@echo off
echo ========================================
echo Building, Fixing, and Pushing Updates
echo ========================================

echo.
echo Step 1: Building locally...
call build_local.bat
if errorlevel 1 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo Step 2: Checking git status...
git status

echo.
echo Step 3: Adding changes...
git add .

echo.
echo Step 4: Committing changes...
set /p commit_msg="Enter commit message: "
git commit -m "%commit_msg%"

echo.
echo Step 5: Pushing to GitHub...
git push origin main

echo.
echo ========================================
echo Process completed successfully!
echo ========================================
pause
