-- PostgreSQL Test Database Initialization Script
-- This script is executed by Docker when the container is first created
-- It creates the test user and database needed for integration tests

-- Create test user if not exists
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'tiris_test') THEN
      CREATE USER tiris_test WITH PASSWORD 'tiris_test' CREATEDB LOGIN;
   END IF;
END
$do$;

-- Create test database if not exists
SELECT 'CREATE DATABASE tiris_test OWNER tiris_test'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'tiris_test')\gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE tiris_test TO tiris_test;

-- Connect to the test database to set up schema privileges
\c tiris_test;

-- Create required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Grant schema privileges
GRANT ALL ON SCHEMA public TO tiris_test;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tiris_test;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO tiris_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO tiris_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO tiris_test;

-- Create a verification table to confirm setup worked
CREATE TABLE IF NOT EXISTS test_setup_verification (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    message TEXT DEFAULT 'Test database setup completed successfully'
);

INSERT INTO test_setup_verification (message) VALUES ('Database initialized by Docker');

-- Display setup completion
SELECT 'Test database setup completed successfully' AS status;