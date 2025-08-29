-- Fix unique constraints to work properly with soft deletion
-- This migration drops existing unique constraints and replaces them with partial unique indexes
-- that only apply to non-deleted records (WHERE deleted_at IS NULL)

-- Check if we need to migrate exchanges or tradings table
DO $$
DECLARE 
    table_to_use TEXT;
    column_to_use TEXT;
BEGIN
    -- Determine which table exists (exchanges or tradings)
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'exchanges') THEN
        table_to_use := 'exchanges';
        column_to_use := 'exchange_id';
        RAISE NOTICE 'Working with exchanges table';
    ELSIF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tradings') THEN
        table_to_use := 'tradings';
        column_to_use := 'trading_id';
        RAISE NOTICE 'Working with tradings table (exchanges already renamed)';
    ELSE
        RAISE NOTICE 'Neither exchanges nor tradings table found - skipping migration';
        RETURN;
    END IF;

    -- Drop existing unique constraints that conflict with soft deletion
    IF table_to_use = 'exchanges' THEN
        -- Drop exchanges unique constraints
        BEGIN
            EXECUTE 'ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_name_unique';
            RAISE NOTICE 'Dropped exchanges_user_name_unique constraint';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'exchanges_user_name_unique constraint does not exist: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_key_unique';
            RAISE NOTICE 'Dropped exchanges_user_api_key_unique constraint';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'exchanges_user_api_key_unique constraint does not exist: %', SQLERRM;
        END;

        BEGIN
            EXECUTE 'ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_secret_unique';
            RAISE NOTICE 'Dropped exchanges_user_api_secret_unique constraint';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'exchanges_user_api_secret_unique constraint does not exist: %', SQLERRM;
        END;

        -- Create partial unique indexes for exchanges table
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_name_active_unique 
                 ON exchanges (user_id, name) 
                 WHERE deleted_at IS NULL';

        -- Only create API key constraints if columns exist (old architecture)
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'exchanges' AND column_name = 'api_key') THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_key_active_unique
                     ON exchanges (user_id, api_key)
                     WHERE deleted_at IS NULL';
        END IF;

        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'exchanges' AND column_name = 'api_secret') THEN
            EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_secret_active_unique
                     ON exchanges (user_id, api_secret)
                     WHERE deleted_at IS NULL';
        END IF;

    ELSE
        -- Working with tradings table
        BEGIN
            EXECUTE 'ALTER TABLE tradings DROP CONSTRAINT IF EXISTS tradings_user_name_unique';
            RAISE NOTICE 'Dropped tradings_user_name_unique constraint';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'tradings_user_name_unique constraint does not exist: %', SQLERRM;
        END;

        -- Create partial unique indexes for tradings table
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS tradings_user_name_active_unique 
                 ON tradings (user_id, name) 
                 WHERE deleted_at IS NULL';
    END IF;

    -- Drop sub_accounts unique constraints
    BEGIN
        EXECUTE 'ALTER TABLE sub_accounts DROP CONSTRAINT IF EXISTS sub_accounts_' || column_to_use || '_name_unique';
        RAISE NOTICE 'Dropped sub_accounts_%_name_unique constraint', column_to_use;
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'sub_accounts_%_name_unique constraint does not exist: %', column_to_use, SQLERRM;
    END;

    -- Create sub-account unique index with correct column name
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sub_accounts' AND column_name = column_to_use) THEN
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS sub_accounts_' || column_to_use || '_name_active_unique
                 ON sub_accounts (' || column_to_use || ', name)
                 WHERE deleted_at IS NULL';
    END IF;

    RAISE NOTICE 'Successfully created partial unique indexes for soft deletion compatibility';
    RAISE NOTICE 'Unique constraints now only apply to active records (deleted_at IS NULL)';
END
$$;