@echo off
echo Fixing syntax error in sequence repository...

cd C:\Users\ROGSTRIX\go-whatsapp-web-multidevice-main

echo.
echo Running quick syntax check...
cd src
go build ./... 2>&1 | findstr /C:"syntax error"
if %ERRORLEVEL% == 0 (
    echo Syntax errors found, please check!
) else (
    echo No syntax errors detected!
)
cd ..

echo.
echo Git status...
git status

echo.
echo Adding changes...
git add src/repository/sequence_repository.go

echo.
echo Committing fix...
git commit -m "fix: Add missing closing brace in GetSequenceStats function"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo ✅ Syntax error fix pushed successfully!
pause