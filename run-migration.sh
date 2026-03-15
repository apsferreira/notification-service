#!/bin/bash

# Run database migration for notification service
# This script assumes PostgreSQL is available via environment variables

set -e

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "DATABASE_URL environment variable is required"
    exit 1
fi

echo "Running OTP table migration..."

# Use psql to run the migration directly
psql "$DATABASE_URL" -f migrations/001_create_otp_codes.up.sql

echo "✅ Migration completed successfully!"