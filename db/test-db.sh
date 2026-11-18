#!/bin/sh
# Creates invoice_test, a second database for the Postgres tests, with the
# same schema. Tests empty its tables, so your dev data in "invoice" is safe.
# Postgres runs this once, on an empty volume, after schema.sql (files run in name order).
set -e
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "CREATE DATABASE invoice_test"
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d invoice_test -f /docker-entrypoint-initdb.d/schema.sql
