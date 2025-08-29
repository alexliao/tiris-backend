-- Add deleted_at columns for GORM soft delete functionality

-- Add deleted_at to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Add deleted_at to oauth_tokens table  
ALTER TABLE oauth_tokens ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_deleted_at ON oauth_tokens(deleted_at);

-- Add deleted_at to tables (conditional based on which architecture is in use)
DO $$
BEGIN
    -- Check if exchanges table exists (old architecture)
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'exchanges') THEN
        -- Old architecture: add to exchanges table
        ALTER TABLE exchanges ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
        CREATE INDEX IF NOT EXISTS idx_exchanges_deleted_at ON exchanges(deleted_at);
    ELSE
        -- New architecture: add to exchange_bindings and tradings tables
        ALTER TABLE exchange_bindings ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
        CREATE INDEX IF NOT EXISTS idx_exchange_bindings_deleted_at ON exchange_bindings(deleted_at);
        
        ALTER TABLE tradings ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
        CREATE INDEX IF NOT EXISTS idx_tradings_deleted_at ON tradings(deleted_at);
    END IF;
END $$;

-- Add deleted_at to sub_accounts table
ALTER TABLE sub_accounts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_sub_accounts_deleted_at ON sub_accounts(deleted_at);

-- Note: transactions and trading_logs are time-series data and should not have soft delete
-- as per the model definitions (they don't include gorm.DeletedAt)