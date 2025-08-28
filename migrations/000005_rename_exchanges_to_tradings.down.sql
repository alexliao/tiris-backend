-- Migration Rollback: Rename tradings back to exchanges
-- This migration reverses the changes made in the up migration

-- Step 1: Drop all foreign key constraints that reference tradings table
ALTER TABLE sub_accounts DROP CONSTRAINT IF EXISTS sub_accounts_trading_id_fkey;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_trading_id_fkey;
ALTER TABLE trading_logs DROP CONSTRAINT IF EXISTS trading_logs_trading_id_fkey;

-- Step 2: Drop triggers on tradings table
DROP TRIGGER IF EXISTS update_tradings_updated_at ON tradings;

-- Step 3: Rename tradings table back to exchanges
ALTER TABLE tradings RENAME TO exchanges;

-- Step 4: Rename columns back in related tables
ALTER TABLE sub_accounts RENAME COLUMN trading_id TO exchange_id;
ALTER TABLE transactions RENAME COLUMN trading_id TO exchange_id;
ALTER TABLE trading_logs RENAME COLUMN trading_id TO exchange_id;

-- Step 5: Drop indexes on tradings table (now exchanges)
DROP INDEX IF EXISTS idx_tradings_user_id;
DROP INDEX IF EXISTS idx_tradings_type;
DROP INDEX IF EXISTS idx_tradings_status;
DROP INDEX IF EXISTS idx_tradings_info;
DROP INDEX IF EXISTS idx_tradings_deleted_at;

-- Step 6: Drop unique indexes for tradings
DROP INDEX IF EXISTS tradings_user_name_active_unique;
DROP INDEX IF EXISTS tradings_user_api_key_active_unique;
DROP INDEX IF EXISTS tradings_user_api_secret_active_unique;

-- Step 7: Create original indexes on exchanges table
CREATE INDEX IF NOT EXISTS idx_exchanges_user_id ON exchanges(user_id);
CREATE INDEX IF NOT EXISTS idx_exchanges_type ON exchanges(type);
CREATE INDEX IF NOT EXISTS idx_exchanges_status ON exchanges(status);
CREATE INDEX IF NOT EXISTS idx_exchanges_info ON exchanges USING GIN(info);
CREATE INDEX IF NOT EXISTS idx_exchanges_deleted_at ON exchanges(deleted_at);

-- Step 8: Recreate original unique indexes for exchanges table
CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_name_active_unique 
ON exchanges (user_id, name) 
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_key_active_unique
ON exchanges (user_id, api_key)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_secret_active_unique
ON exchanges (user_id, api_secret)
WHERE deleted_at IS NULL;

-- Step 9: Drop indexes on related tables that reference trading_id
DROP INDEX IF EXISTS idx_sub_accounts_trading_id;
DROP INDEX IF EXISTS idx_transactions_trading_id_timestamp;
DROP INDEX IF EXISTS idx_trading_logs_trading_id_timestamp;
DROP INDEX IF EXISTS sub_accounts_trading_name_active_unique;

-- Step 10: Recreate original indexes on related tables with exchange_id
CREATE INDEX IF NOT EXISTS idx_sub_accounts_exchange_id ON sub_accounts(exchange_id);
CREATE INDEX IF NOT EXISTS idx_transactions_exchange_id_timestamp ON transactions(exchange_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trading_logs_exchange_id_timestamp ON trading_logs(exchange_id, timestamp DESC);
CREATE UNIQUE INDEX IF NOT EXISTS sub_accounts_exchange_name_active_unique
ON sub_accounts (exchange_id, name)
WHERE deleted_at IS NULL;

-- Step 11: Re-create original foreign key constraints
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_exchange_id_fkey 
    FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE;

ALTER TABLE transactions ADD CONSTRAINT transactions_exchange_id_fkey 
    FOREIGN KEY (exchange_id) REFERENCES exchanges(id);

ALTER TABLE trading_logs ADD CONSTRAINT trading_logs_exchange_id_fkey 
    FOREIGN KEY (exchange_id) REFERENCES exchanges(id);

-- Step 12: Recreate original trigger for exchanges table
CREATE OR REPLACE TRIGGER update_exchanges_updated_at BEFORE UPDATE ON exchanges 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Step 13: Restore original balance update function
CREATE OR REPLACE FUNCTION update_sub_account_balance(
    p_sub_account_id UUID,
    p_new_balance DECIMAL(20,8),
    p_amount DECIMAL(20,8),
    p_direction VARCHAR(10),
    p_reason VARCHAR(50),
    p_info JSONB DEFAULT '{}'
)
RETURNS UUID AS $$
DECLARE
    v_user_id UUID;
    v_exchange_id UUID;
    v_transaction_id UUID;
BEGIN
    -- Get user_id and exchange_id from sub_account
    SELECT user_id, exchange_id INTO v_user_id, v_exchange_id
    FROM sub_accounts 
    WHERE id = p_sub_account_id;
    
    IF NOT FOUND THEN
        RAISE EXCEPTION 'Sub-account not found: %', p_sub_account_id;
    END IF;
    
    -- Update the sub-account balance
    UPDATE sub_accounts 
    SET balance = p_new_balance, updated_at = NOW()
    WHERE id = p_sub_account_id;
    
    -- Create transaction record
    INSERT INTO transactions (
        user_id, exchange_id, sub_account_id, direction, reason, 
        amount, closing_balance, info
    ) VALUES (
        v_user_id, v_exchange_id, p_sub_account_id, p_direction, p_reason,
        p_amount, p_new_balance, p_info
    ) RETURNING id INTO v_transaction_id;
    
    RETURN v_transaction_id;
END;
$$ LANGUAGE plpgsql;

-- Add notification about successful rollback
DO $$
BEGIN
    RAISE NOTICE 'Successfully rolled back tradings table to exchanges and restored all references';
END
$$;