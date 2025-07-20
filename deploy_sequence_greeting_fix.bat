@echo off
echo === SEQUENCE GREETING FIX DEPLOYMENT ===
echo.
echo Fixing sequence messages to include Malaysian greetings...
echo.

REM Build the application
echo Building application...
call build_local.bat

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo Build successful!
echo.
echo === CHANGES APPLIED ===
echo - Sequence messages will now include Malaysian greetings (Hi/Hello/Salam + name)
echo - Proper line breaks between greeting and message content
echo - Recipient name handling (shows "Cik" if name is missing)
echo.
echo Please restart the application for changes to take effect.
echo.
pause
