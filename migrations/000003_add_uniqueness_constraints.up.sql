-- Add uniqueness constraints to ensure data integrity

-- Add unique constraint for exchange names per user
-- This ensures each user cannot have multiple exchanges with the same name
ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_name_unique UNIQUE (user_id, name);

-- Add unique constraint for API keys per user
-- This prevents users from accidentally using the same API key for multiple exchanges
ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_key_unique UNIQUE (user_id, api_key);

-- Add unique constraint for API secrets per user  
-- This prevents users from accidentally using the same API secret for multiple exchanges
ALTER TABLE exchanges ADD CONSTRAINT exchanges_user_api_secret_unique UNIQUE (user_id, api_secret);

-- Add unique constraint for sub-account names per exchange
-- This ensures each exchange cannot have multiple sub-accounts with the same name
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_exchange_name_unique UNIQUE (exchange_id, name);