-- Drop function
DROP FUNCTION IF EXISTS update_sub_account_balance(UUID, DECIMAL(20,8), DECIMAL(20,8), VARCHAR(10), VARCHAR(50), JSONB);

-- Drop triggers
DROP TRIGGER IF EXISTS update_sub_accounts_updated_at ON sub_accounts;
DROP TRIGGER IF EXISTS update_exchanges_updated_at ON exchanges;
DROP TRIGGER IF EXISTS update_oauth_tokens_updated_at ON oauth_tokens;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS event_processing;
DROP TABLE IF EXISTS trading_logs;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS sub_accounts;
DROP TABLE IF EXISTS exchanges;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS users;

-- Drop extensions (only if no other objects depend on them)
-- Note: Be careful with these in production as other databases might depend on them
-- DROP EXTENSION IF EXISTS timescaledb CASCADE;
-- DROP EXTENSION IF EXISTS "uuid-ossp";