-- Remove deleted_at columns (rollback migration)

-- Remove from sub_accounts table
DROP INDEX IF EXISTS idx_sub_accounts_deleted_at;
ALTER TABLE sub_accounts DROP COLUMN IF EXISTS deleted_at;

-- Remove from exchanges table
DROP INDEX IF EXISTS idx_exchanges_deleted_at;
ALTER TABLE exchanges DROP COLUMN IF EXISTS deleted_at;

-- Remove from oauth_tokens table
DROP INDEX IF EXISTS idx_oauth_tokens_deleted_at;
ALTER TABLE oauth_tokens DROP COLUMN IF EXISTS deleted_at;

-- Remove from users table
DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;