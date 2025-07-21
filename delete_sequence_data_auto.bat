@echo off
echo ================================================
echo Deleting All Sequence Data
echo ================================================
echo.
echo Deleting sequence data...

REM Set password for PostgreSQL (update if needed)
set PGPASSWORD=password

echo.
echo Connecting to database and deleting data...
psql -U postgres -d whatsapp_db -f delete_all_sequence_data.sql

IF %ERRORLEVEL% EQU 0 (
    echo.
    echo ================================================
    echo Sequence data deletion complete!
    echo ================================================
) ELSE (
    echo.
    echo ================================================
    echo ERROR: Failed to delete sequence data
    echo Please check your database connection
    echo ================================================
)
