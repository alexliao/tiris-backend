-- Migration: Rename exchanges to tradings (Legacy migration - may be skipped if schema is already updated)
-- This migration renames the exchanges table to tradings and updates all references

DO $$
BEGIN
    -- Check if we're on the new architecture already
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tradings') THEN
        RAISE NOTICE 'Tradings table already exists - skipping exchanges rename migration';
        RETURN;
    END IF;
    
    -- Check if exchanges table exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'exchanges') THEN
        RAISE NOTICE 'Exchanges table does not exist - skipping migration';
        RETURN;
    END IF;
    
    -- Step 1: Drop all foreign key constraints that reference exchanges table
    ALTER TABLE sub_accounts DROP CONSTRAINT IF EXISTS sub_accounts_exchange_id_fkey;
    ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_exchange_id_fkey;
    ALTER TABLE trading_logs DROP CONSTRAINT IF EXISTS trading_logs_exchange_id_fkey;

    -- Step 2: Drop triggers on exchanges table
    DROP TRIGGER IF EXISTS update_exchanges_updated_at ON exchanges;

    -- Step 3: Rename exchanges table to tradings
    ALTER TABLE exchanges RENAME TO tradings;

    -- Step 4: Rename columns in related tables
    ALTER TABLE sub_accounts RENAME COLUMN exchange_id TO trading_id;
    ALTER TABLE transactions RENAME COLUMN exchange_id TO trading_id;
    ALTER TABLE trading_logs RENAME COLUMN exchange_id TO trading_id;

    -- Step 5: Drop old indexes on exchanges table (now tradings)
    DROP INDEX IF EXISTS idx_exchanges_user_id;
    DROP INDEX IF EXISTS idx_exchanges_type;
    DROP INDEX IF EXISTS idx_exchanges_status;
    DROP INDEX IF EXISTS idx_exchanges_info;
    DROP INDEX IF EXISTS idx_exchanges_deleted_at;

    -- Step 6: Drop old unique indexes
    DROP INDEX IF EXISTS exchanges_user_name_active_unique;
    DROP INDEX IF EXISTS exchanges_user_api_key_active_unique;
    DROP INDEX IF EXISTS exchanges_user_api_secret_active_unique;

    -- Step 7: Create new indexes on tradings table
    CREATE INDEX IF NOT EXISTS idx_tradings_user_id ON tradings(user_id);
    CREATE INDEX IF NOT EXISTS idx_tradings_type ON tradings(type);
    CREATE INDEX IF NOT EXISTS idx_tradings_status ON tradings(status);
    CREATE INDEX IF NOT EXISTS idx_tradings_info ON tradings USING GIN(info);
    CREATE INDEX IF NOT EXISTS idx_tradings_deleted_at ON tradings(deleted_at);

    -- Step 8: Create new unique indexes for tradings table (without API key constraints)
    CREATE UNIQUE INDEX IF NOT EXISTS tradings_user_name_active_unique 
    ON tradings (user_id, name) 
    WHERE deleted_at IS NULL;

    -- Step 9: Drop old indexes on related tables that reference exchange_id
    DROP INDEX IF EXISTS idx_sub_accounts_exchange_id;
    DROP INDEX IF EXISTS idx_transactions_exchange_id_timestamp;
    DROP INDEX IF EXISTS idx_trading_logs_exchange_id_timestamp;
    DROP INDEX IF EXISTS sub_accounts_exchange_name_active_unique;

    -- Step 10: Create new indexes on related tables with trading_id
    CREATE INDEX IF NOT EXISTS idx_sub_accounts_trading_id ON sub_accounts(trading_id);
    CREATE INDEX IF NOT EXISTS idx_transactions_trading_id_timestamp ON transactions(trading_id, timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_trading_logs_trading_id_timestamp ON trading_logs(trading_id, timestamp DESC);
    CREATE UNIQUE INDEX IF NOT EXISTS sub_accounts_trading_name_active_unique
    ON sub_accounts (trading_id, name)
    WHERE deleted_at IS NULL;

    -- Step 11: Re-create foreign key constraints with new column names
    ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_trading_id_fkey 
        FOREIGN KEY (trading_id) REFERENCES tradings(id) ON DELETE CASCADE;

    ALTER TABLE transactions ADD CONSTRAINT transactions_trading_id_fkey 
        FOREIGN KEY (trading_id) REFERENCES tradings(id);

    ALTER TABLE trading_logs ADD CONSTRAINT trading_logs_trading_id_fkey 
        FOREIGN KEY (trading_id) REFERENCES tradings(id);

    -- Step 12: Create new trigger for tradings table
    CREATE OR REPLACE TRIGGER update_tradings_updated_at BEFORE UPDATE ON tradings 
        FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    
    RAISE NOTICE 'Successfully renamed exchanges table to tradings and updated all references';

END
$$;

-- Update the balance update function to use new table and column names (if needed)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.routines WHERE routine_name = 'update_sub_account_balance') THEN
        CREATE OR REPLACE FUNCTION update_sub_account_balance(
            p_sub_account_id UUID,
            p_new_balance DECIMAL(20,8),
            p_amount DECIMAL(20,8),
            p_direction VARCHAR(10),
            p_reason VARCHAR(50),
            p_info JSONB DEFAULT '{}'
        )
        RETURNS UUID AS $func$
        DECLARE
            v_user_id UUID;
            v_trading_id UUID;
            v_transaction_id UUID;
        BEGIN
            -- Get user_id and trading_id from sub_account
            SELECT user_id, trading_id INTO v_user_id, v_trading_id
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
                user_id, trading_id, sub_account_id, direction, reason, 
                amount, closing_balance, info
            ) VALUES (
                v_user_id, v_trading_id, p_sub_account_id, p_direction, p_reason,
                p_amount, p_new_balance, p_info
            ) RETURNING id INTO v_transaction_id;
            
            RETURN v_transaction_id;
        END;
        $func$ LANGUAGE plpgsql;
    END IF;
END
$$;