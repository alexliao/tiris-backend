-- Migration rollback: Remove exchange bindings architecture
-- This migration rolls back the exchange bindings separation

-- Step 1: Add API credential columns back to tradings table
ALTER TABLE tradings ADD COLUMN IF NOT EXISTS api_key TEXT;
ALTER TABLE tradings ADD COLUMN IF NOT EXISTS api_secret TEXT;

-- Step 2: Migrate API credentials back to tradings table
UPDATE tradings SET 
    api_key = eb.api_key,
    api_secret = eb.api_secret
FROM exchange_bindings eb
WHERE tradings.exchange_binding_id = eb.id
AND eb.type = 'private';

-- Step 3: Drop foreign key constraint
ALTER TABLE tradings DROP CONSTRAINT IF EXISTS tradings_exchange_binding_id_fkey;

-- Step 4: Drop exchange_binding_id column
ALTER TABLE tradings DROP COLUMN IF EXISTS exchange_binding_id;

-- Step 5: Drop exchange_binding_id index
DROP INDEX IF EXISTS idx_tradings_exchange_binding_id;

-- Step 6: Re-create old unique constraints on API keys
CREATE UNIQUE INDEX IF NOT EXISTS tradings_user_api_key_active_unique
ON tradings (user_id, api_key)
WHERE deleted_at IS NULL AND api_key IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS tradings_user_api_secret_active_unique
ON tradings (user_id, api_secret)
WHERE deleted_at IS NULL AND api_secret IS NOT NULL;

-- Step 7: Drop exchange bindings table and related objects
DROP TRIGGER IF EXISTS update_exchange_bindings_updated_at ON exchange_bindings;
DROP TABLE IF EXISTS exchange_bindings CASCADE;

-- Step 8: Add notification
DO $$
BEGIN
    RAISE NOTICE 'Successfully rolled back exchange bindings architecture';
END
$$;