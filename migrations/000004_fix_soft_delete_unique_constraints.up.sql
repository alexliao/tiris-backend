-- Fix unique constraints to work properly with soft deletion
-- This migration drops existing unique constraints and replaces them with partial unique indexes
-- that only apply to non-deleted records (WHERE deleted_at IS NULL)

-- Drop existing unique constraints that conflict with soft deletion
DO $$
BEGIN
    -- Drop exchanges unique constraints
    ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_name_unique;
    RAISE NOTICE 'Dropped exchanges_user_name_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'exchanges_user_name_unique constraint does not exist or could not be dropped: %', SQLERRM;
END
$$;

DO $$
BEGIN
    ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_key_unique;
    RAISE NOTICE 'Dropped exchanges_user_api_key_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'exchanges_user_api_key_unique constraint does not exist or could not be dropped: %', SQLERRM;
END
$$;

DO $$
BEGIN
    ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_secret_unique;
    RAISE NOTICE 'Dropped exchanges_user_api_secret_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'exchanges_user_api_secret_unique constraint does not exist or could not be dropped: %', SQLERRM;
END
$$;

-- Drop sub_accounts unique constraints
DO $$
BEGIN
    ALTER TABLE sub_accounts DROP CONSTRAINT IF EXISTS sub_accounts_exchange_name_unique;
    RAISE NOTICE 'Dropped sub_accounts_exchange_name_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'sub_accounts_exchange_name_unique constraint does not exist or could not be dropped: %', SQLERRM;
END
$$;

-- Create partial unique indexes that exclude soft-deleted records
-- These indexes enforce uniqueness only on active records (deleted_at IS NULL)

-- Exchange name uniqueness per user (only for active records)
CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_name_active_unique 
ON exchanges (user_id, name) 
WHERE deleted_at IS NULL;

-- API key uniqueness per user (only for active records)  
CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_key_active_unique
ON exchanges (user_id, api_key)
WHERE deleted_at IS NULL;

-- API secret uniqueness per user (only for active records)
CREATE UNIQUE INDEX IF NOT EXISTS exchanges_user_api_secret_active_unique
ON exchanges (user_id, api_secret)
WHERE deleted_at IS NULL;

-- Sub-account name uniqueness per exchange (only for active records)
CREATE UNIQUE INDEX IF NOT EXISTS sub_accounts_exchange_name_active_unique
ON sub_accounts (exchange_id, name)
WHERE deleted_at IS NULL;

-- Log successful completion
DO $$
BEGIN
    RAISE NOTICE 'Successfully created partial unique indexes for soft deletion compatibility';
    RAISE NOTICE 'Unique constraints now only apply to active records (deleted_at IS NULL)';
END
$$;