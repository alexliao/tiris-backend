-- Database initialization script for TimescaleDB
-- This script sets up the database with TimescaleDB extension and basic configuration

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable crypto functions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Set timezone to UTC
SET timezone = 'UTC';

-- Create database if it doesn't exist (this handles fresh installations)
SELECT 'CREATE DATABASE tiris_dev OWNER tiris_user' 
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'tiris_dev')\gexec

-- Create development user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tiris_user') THEN
        CREATE USER tiris_user WITH PASSWORD 'tiris_password';
    END IF;
END
$$;

-- Grant privileges to development user
GRANT ALL PRIVILEGES ON DATABASE tiris_dev TO tiris_user;
GRANT ALL ON SCHEMA public TO tiris_user;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO tiris_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO tiris_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO tiris_user;

-- Create application schema if needed
CREATE SCHEMA IF NOT EXISTS tiris AUTHORIZATION tiris_user;

-- Set search path to include both schemas
ALTER ROLE tiris_user SET search_path TO tiris, public;

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE 'TimescaleDB development database initialized successfully';
    RAISE NOTICE 'Database: tiris_dev';
    RAISE NOTICE 'User: tiris_user';
    RAISE NOTICE 'Extensions: timescaledb, uuid-ossp, pgcrypto';
END
$$;