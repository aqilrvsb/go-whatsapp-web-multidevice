#!/bin/sh

# Railway provides DATABASE_URL, but our app expects DB_URI
if [ -n "$DATABASE_URL" ]; then
    export DB_URI="$DATABASE_URL"
    echo "✅ DB_URI set from DATABASE_URL"
else
    echo "⚠️  DATABASE_URL not found"
fi

# Log environment for debugging
echo "Environment variables:"
echo "PORT: ${PORT:-not set}"
echo "DB_URI: ${DB_URI:-not set}"
echo "APP_PORT: ${APP_PORT:-not set}"

# Start the application
echo "Starting WhatsApp Multi-Device..."
exec /app/whatsapp
