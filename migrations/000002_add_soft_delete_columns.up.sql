-- Add deleted_at columns for GORM soft delete functionality

-- Add deleted_at to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Add deleted_at to oauth_tokens table  
ALTER TABLE oauth_tokens ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_oauth_tokens_deleted_at ON oauth_tokens(deleted_at);

-- Add deleted_at to exchanges table
ALTER TABLE exchanges ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_exchanges_deleted_at ON exchanges(deleted_at);

-- Add deleted_at to sub_accounts table
ALTER TABLE sub_accounts ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX idx_sub_accounts_deleted_at ON sub_accounts(deleted_at);

-- Note: transactions and trading_logs are time-series data and should not have soft delete
-- as per the model definitions (they don't include gorm.DeletedAt)