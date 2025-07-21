@echo off
echo ================================================
echo Deleting All Sequence Data
echo ================================================
echo.
echo WARNING: This will permanently delete all sequence data!
echo.

REM Set PostgreSQL password
set PGPASSWORD=password

echo Attempting to connect to PostgreSQL...
echo.

REM Try to run psql with the connection string
psql "postgresql://postgres:password@localhost:5432/whatsapp_db?sslmode=disable" -f delete_all_sequence_data.sql

IF %ERRORLEVEL% EQU 0 (
    echo.
    echo ================================================
    echo Sequence data deletion complete!
    echo ================================================
) ELSE (
    echo.
    echo ================================================
    echo ERROR: Failed to delete sequence data
    echo.
    echo Possible issues:
    echo 1. PostgreSQL is not installed or psql is not in PATH
    echo 2. PostgreSQL service is not running
    echo 3. Database credentials are incorrect
    echo.
    echo To install psql on Windows:
    echo - Download PostgreSQL from https://www.postgresql.org/download/windows/
    echo - Or use a GUI tool like pgAdmin, DBeaver, or TablePlus
    echo ================================================
)
pause