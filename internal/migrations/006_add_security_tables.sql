-- Migration: Add security tables and update existing tables for enhanced security
-- Description: This migration adds tables for audit events, user API keys, and updates
-- the exchanges table to use encrypted API keys

-- Create audit_events table for security logging
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(20) NOT NULL,
    action VARCHAR(50) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id VARCHAR(255),
    ip_address INET NOT NULL,
    user_agent TEXT,
    resource VARCHAR(255),
    details JSONB DEFAULT '{}',
    success BOOLEAN NOT NULL DEFAULT true,
    error TEXT,
    duration BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for audit_events
CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_level ON audit_events(level);
CREATE INDEX idx_audit_events_action ON audit_events(action);
CREATE INDEX idx_audit_events_user_id ON audit_events(user_id);
CREATE INDEX idx_audit_events_session_id ON audit_events(session_id);
CREATE INDEX idx_audit_events_ip_address ON audit_events(ip_address);
CREATE INDEX idx_audit_events_resource ON audit_events(resource);
CREATE INDEX idx_audit_events_success ON audit_events(success);

-- Create composite indexes for common queries
CREATE INDEX idx_audit_events_user_timestamp ON audit_events(user_id, timestamp DESC);
CREATE INDEX idx_audit_events_action_timestamp ON audit_events(action, timestamp DESC);
CREATE INDEX idx_audit_events_level_timestamp ON audit_events(level, timestamp DESC) WHERE level IN ('warn', 'error', 'critical');

-- Create user_api_keys table for API key management
CREATE TABLE IF NOT EXISTS user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    encrypted_key TEXT NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for user_api_keys
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX idx_user_api_keys_key_hash ON user_api_keys(key_hash);
CREATE INDEX idx_user_api_keys_is_active ON user_api_keys(is_active);
CREATE INDEX idx_user_api_keys_expires_at ON user_api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Create unique constraint for active API keys per user
CREATE UNIQUE INDEX idx_user_api_keys_unique_hash ON user_api_keys(key_hash) WHERE is_active = true;

-- Add new security columns to exchanges table
DO $$ 
BEGIN
    -- Add encrypted_api_key column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'encrypted_api_key'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN encrypted_api_key TEXT;
    END IF;

    -- Add encrypted_api_secret column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'encrypted_api_secret'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN encrypted_api_secret TEXT;
    END IF;

    -- Add api_key_hash column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'api_key_hash'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN api_key_hash VARCHAR(64);
    END IF;

    -- Add last_used_at column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'last_used_at'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN last_used_at TIMESTAMPTZ;
    END IF;

    -- Add failure_count column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'failure_count'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN failure_count INTEGER DEFAULT 0;
    END IF;

    -- Add last_failure_at column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'last_failure_at'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN last_failure_at TIMESTAMPTZ;
    END IF;

    -- Add security_settings column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'security_settings'
    ) THEN
        ALTER TABLE exchanges ADD COLUMN security_settings JSONB DEFAULT '{}';
    END IF;
END $$;

-- Create index for api_key_hash if new columns were added
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'exchanges' AND column_name = 'api_key_hash'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_exchanges_api_key_hash ON exchanges(api_key_hash);
    END IF;
END $$;

-- Add constraints for security columns
DO $$
BEGIN
    -- Add check constraint for failure_count
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'exchanges' AND constraint_name = 'chk_exchanges_failure_count'
    ) THEN
        ALTER TABLE exchanges ADD CONSTRAINT chk_exchanges_failure_count CHECK (failure_count >= 0);
    END IF;
END $$;

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
DROP TRIGGER IF EXISTS update_user_api_keys_updated_at ON user_api_keys;
CREATE TRIGGER update_user_api_keys_updated_at 
    BEFORE UPDATE ON user_api_keys 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create partial indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_exchanges_active ON exchanges(user_id, status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_exchanges_last_used ON exchanges(last_used_at DESC) WHERE last_used_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_exchanges_failures ON exchanges(failure_count DESC) WHERE failure_count > 0;

-- Create view for security dashboard (optional)
CREATE OR REPLACE VIEW security_summary AS
SELECT 
    DATE_TRUNC('hour', timestamp) as hour,
    level,
    action,
    COUNT(*) as event_count,
    COUNT(DISTINCT user_id) as unique_users,
    COUNT(DISTINCT ip_address) as unique_ips
FROM audit_events 
WHERE timestamp >= NOW() - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', timestamp), level, action
ORDER BY hour DESC, event_count DESC;

-- Add comments for documentation
COMMENT ON TABLE audit_events IS 'Security audit log for all system events';
COMMENT ON TABLE user_api_keys IS 'User-generated API keys for programmatic access';
COMMENT ON COLUMN exchanges.encrypted_api_key IS 'AES-256-GCM encrypted exchange API key';
COMMENT ON COLUMN exchanges.encrypted_api_secret IS 'AES-256-GCM encrypted exchange API secret';
COMMENT ON COLUMN exchanges.api_key_hash IS 'SHA-256 hash of API key for validation';
COMMENT ON COLUMN exchanges.security_settings IS 'Security configuration for exchange access';

-- Insert initial audit event for migration
INSERT INTO audit_events (
    level, 
    action, 
    ip_address, 
    user_agent, 
    resource, 
    details, 
    success
) VALUES (
    'info',
    'system.migration',
    '127.0.0.1',
    'Database Migration Script',
    'database',
    '{"migration": "006_add_security_tables", "description": "Added security tables and enhanced exchange security"}',
    true
);

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON audit_events TO tiris_app_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON user_api_keys TO tiris_app_user;
-- GRANT SELECT ON security_summary TO tiris_app_user;