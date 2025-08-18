-- Rollback uniqueness constraints

-- Remove unique constraint for sub-account names per exchange
ALTER TABLE sub_accounts DROP CONSTRAINT IF EXISTS sub_accounts_exchange_name_unique;

-- Remove unique constraint for API secrets per user
ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_secret_unique;

-- Remove unique constraint for API keys per user  
ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_api_key_unique;

-- Remove unique constraint for exchange names per user
ALTER TABLE exchanges DROP CONSTRAINT IF EXISTS exchanges_user_name_unique;