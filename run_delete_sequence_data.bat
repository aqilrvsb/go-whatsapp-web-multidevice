@echo off
echo ================================================
echo Deleting All Sequence Data
echo ================================================
echo.
echo WARNING: This will permanently delete:
echo - All sequences
echo - All sequence steps
echo - All sequence contacts (enrollments)
echo - All sequence-related messages
echo.
echo Press Ctrl+C to cancel or
pause

echo.
echo Connecting to database...
psql -U postgres -d whatsapp_db -f delete_all_sequence_data.sql

echo.
echo ================================================
echo Sequence data deletion complete!
echo ================================================
pause
