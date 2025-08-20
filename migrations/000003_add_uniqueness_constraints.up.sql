-- Add uniqueness constraints to ensure data integrity

-- Add unique constraint for exchange names per user (conditional)
-- This ensures each user cannot have multiple exchanges with the same name
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_name_unique UNIQUE (user_id, name);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Constraint exchanges_user_name_unique already exists or failed to create';
END
$$;

-- Add unique constraint for API keys per user (conditional)
-- This prevents users from accidentally using the same API key for multiple exchanges
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_key_unique UNIQUE (user_id, api_key);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Constraint exchanges_user_api_key_unique already exists or failed to create';
END
$$;

-- Add unique constraint for API secrets per user (conditional)
-- This prevents users from accidentally using the same API secret for multiple exchanges
DO $$
BEGIN
    ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_secret_unique UNIQUE (user_id, api_secret);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Constraint exchanges_user_api_secret_unique already exists or failed to create';
END
$$;

-- Add unique constraint for sub-account names per exchange (conditional)
-- This ensures each exchange cannot have multiple sub-accounts with the same name
DO $$
BEGIN
    ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_exchange_name_unique UNIQUE (exchange_id, name);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Constraint sub_accounts_exchange_name_unique already exists or failed to create';
END
$$;