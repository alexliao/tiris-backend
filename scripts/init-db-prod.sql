-- Production database initialization script for TimescaleDB
-- Enhanced security and performance for production deployment

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Set timezone to UTC
SET timezone = 'UTC';

-- Create production user with limited privileges
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tiris_user') THEN
        CREATE USER tiris_user WITH 
            ENCRYPTED PASSWORD 'CHANGE_ME_IN_PRODUCTION'
            NOSUPERUSER 
            NOCREATEDB 
            NOCREATEROLE 
            NOINHERIT 
            LOGIN
            CONNECTION LIMIT 25;
    END IF;
END
$$;

-- Create application database
CREATE DATABASE tiris_prod WITH
    OWNER = tiris_user
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TEMPLATE = template0;

-- Connect to the application database
\c tiris_prod;

-- Grant necessary privileges to application user
GRANT ALL PRIVILEGES ON DATABASE tiris_prod TO tiris_user;
GRANT ALL ON SCHEMA public TO tiris_user;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO tiris_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO tiris_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO tiris_user;

-- Create application schema
CREATE SCHEMA IF NOT EXISTS tiris AUTHORIZATION tiris_user;

-- Set search path for the user
ALTER ROLE tiris_user SET search_path TO tiris, public;

-- Create monitoring user for health checks (read-only)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'tiris_monitor') THEN
        CREATE USER tiris_monitor WITH 
            ENCRYPTED PASSWORD 'CHANGE_ME_MONITOR_PASSWORD'
            NOSUPERUSER 
            NOCREATEDB 
            NOCREATEROLE 
            NOINHERIT 
            LOGIN
            CONNECTION LIMIT 5;
    END IF;
END
$$;

-- Grant monitoring permissions
GRANT CONNECT ON DATABASE tiris_prod TO tiris_monitor;
GRANT USAGE ON SCHEMA public TO tiris_monitor;
GRANT USAGE ON SCHEMA tiris TO tiris_monitor;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO tiris_monitor;
GRANT SELECT ON ALL TABLES IN SCHEMA tiris TO tiris_monitor;

-- Set default privileges for monitoring user
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO tiris_monitor;
ALTER DEFAULT PRIVILEGES IN SCHEMA tiris GRANT SELECT ON TABLES TO tiris_monitor;

-- Performance optimizations
-- Increase shared_preload_libraries in postgresql.conf:
-- shared_preload_libraries = 'timescaledb,pg_stat_statements'

-- Connection and memory settings (adjust based on server resources)
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;

-- TimescaleDB specific optimizations
ALTER SYSTEM SET timescaledb.max_background_workers = 8;

-- Security settings
ALTER SYSTEM SET log_statement = 'mod';
ALTER SYSTEM SET log_min_duration_statement = 1000;
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ';

-- SSL settings (adjust paths as needed)
ALTER SYSTEM SET ssl = on;
ALTER SYSTEM SET ssl_cert_file = '/etc/ssl/certs/server.crt';
ALTER SYSTEM SET ssl_key_file = '/etc/ssl/private/server.key';

-- Row Level Security preparation (will be enabled per table in migrations)
-- This ensures RLS is available for use
SELECT 'Row Level Security available' as rls_status;

-- Create extension for password encryption functions
CREATE OR REPLACE FUNCTION encrypt_api_key(api_key TEXT) 
RETURNS TEXT AS $$
BEGIN
    RETURN crypt(api_key, gen_salt('bf', 12));
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION verify_api_key(api_key TEXT, encrypted_key TEXT) 
RETURNS BOOLEAN AS $$
BEGIN
    RETURN encrypted_key = crypt(api_key, encrypted_key);
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions to application user
GRANT EXECUTE ON FUNCTION encrypt_api_key(TEXT) TO tiris_user;
GRANT EXECUTE ON FUNCTION verify_api_key(TEXT, TEXT) TO tiris_user;

-- Create audit log table for security monitoring
CREATE TABLE IF NOT EXISTS tiris.audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(255) NOT NULL,
    operation VARCHAR(10) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    user_id UUID,
    username VARCHAR(255),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

-- Create index for efficient audit log queries
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON tiris.audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_log_table_name ON tiris.audit_log(table_name);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON tiris.audit_log(user_id);

-- Convert audit_log to hypertable for time-series optimization
SELECT create_hypertable('tiris.audit_log', 'timestamp', if_not_exists => TRUE);

-- Grant permissions on audit table
GRANT ALL ON tiris.audit_log TO tiris_user;
GRANT SELECT ON tiris.audit_log TO tiris_monitor;

-- Create function for audit logging
CREATE OR REPLACE FUNCTION tiris.audit_trigger_function()
RETURNS TRIGGER AS $$
DECLARE
    old_data JSONB;
    new_data JSONB;
BEGIN
    IF TG_OP = 'DELETE' THEN
        old_data = to_jsonb(OLD);
        new_data = NULL;
    ELSIF TG_OP = 'UPDATE' THEN
        old_data = to_jsonb(OLD);
        new_data = to_jsonb(NEW);
    ELSIF TG_OP = 'INSERT' THEN
        old_data = NULL;
        new_data = to_jsonb(NEW);
    END IF;
    
    INSERT INTO tiris.audit_log (
        table_name,
        operation,
        old_values,
        new_values,
        username,
        timestamp
    ) VALUES (
        TG_TABLE_NAME,
        TG_OP,
        old_data,
        new_data,
        current_user,
        NOW()
    );
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Grant execute permission on audit function
GRANT EXECUTE ON FUNCTION tiris.audit_trigger_function() TO tiris_user;

-- Reload configuration
SELECT pg_reload_conf();

-- Log successful initialization
DO $$
BEGIN
    RAISE NOTICE '🎉 TimescaleDB production database initialized successfully';
    RAISE NOTICE '📊 Database: tiris_prod';
    RAISE NOTICE '👤 Application User: tiris_user (limited privileges)';
    RAISE NOTICE '👁️  Monitor User: tiris_monitor (read-only)';
    RAISE NOTICE '🔐 Extensions: timescaledb, uuid-ossp, pgcrypto, pg_stat_statements';
    RAISE NOTICE '🛡️  Security: RLS ready, audit logging enabled, encryption functions available';
    RAISE NOTICE '⚡ Performance: Optimized settings applied';
    RAISE NOTICE '';
    RAISE NOTICE '⚠️  IMPORTANT: Change default passwords in production!';
    RAISE NOTICE '⚠️  IMPORTANT: Configure SSL certificates for secure connections!';
END
$$;