-- Rollback migration: Restore original unique constraints and drop partial indexes
-- This migration restores the original behavior where unique constraints apply to all records
-- WARNING: This rollback may fail if there are soft-deleted records with duplicate unique values

-- Drop the partial unique indexes created in the up migration
DROP INDEX IF EXISTS exchanges_user_name_active_unique;
DROP INDEX IF EXISTS exchanges_user_api_key_active_unique;  
DROP INDEX IF EXISTS exchanges_user_api_secret_active_unique;
DROP INDEX IF EXISTS sub_accounts_exchange_name_active_unique;

-- Recreate the original unique constraints
-- NOTE: These may fail if soft-deleted records have duplicate values

-- Add unique constraint for exchange names per user
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_name_unique UNIQUE (user_id, name);
    RAISE NOTICE 'Recreated exchanges_user_name_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Failed to recreate exchanges_user_name_unique constraint: %', SQLERRM;
    RAISE NOTICE 'This may be due to existing soft-deleted records with duplicate values';
END
$$;

-- Add unique constraint for API keys per user
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_key_unique UNIQUE (user_id, api_key);
    RAISE NOTICE 'Recreated exchanges_user_api_key_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Failed to recreate exchanges_user_api_key_unique constraint: %', SQLERRM;
    RAISE NOTICE 'This may be due to existing soft-deleted records with duplicate values';
END
$$;

-- Add unique constraint for API secrets per user
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_secret_unique UNIQUE (user_id, api_secret);
    RAISE NOTICE 'Recreated exchanges_user_api_secret_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Failed to recreate exchanges_user_api_secret_unique constraint: %', SQLERRM;
    RAISE NOTICE 'This may be due to existing soft-deleted records with duplicate values';
END
$$;

-- Add unique constraint for sub-account names per exchange
DO $$
BEGIN
    ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_exchange_name_unique UNIQUE (exchange_id, name);
    RAISE NOTICE 'Recreated sub_accounts_exchange_name_unique constraint';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Failed to recreate sub_accounts_exchange_name_unique constraint: %', SQLERRM;
    RAISE NOTICE 'This may be due to existing soft-deleted records with duplicate values';
END
$$;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Rollback completed - restored original unique constraints';
    RAISE NOTICE 'Note: Some constraints may have failed due to existing soft-deleted duplicates';
END
$$;