-- Migration: Add exchange bindings architecture
-- This migration updates the existing schema to separate exchange credentials from trading configurations

-- Step 1: Check if we're starting from the new architecture or need to migrate
-- If tradings table already has exchange_binding_id, we're good
DO $$
BEGIN
    -- Check if tradings.exchange_binding_id exists
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tradings' 
        AND column_name = 'exchange_binding_id'
    ) THEN
        RAISE NOTICE 'Exchange bindings architecture already exists - skipping migration';
        RETURN;
    END IF;
    
    -- If we get here, we need to migrate from old architecture
    RAISE NOTICE 'Migrating to exchange bindings architecture';
    
    -- Step 2: Create exchange_bindings table if it doesn't exist
    CREATE TABLE IF NOT EXISTS exchange_bindings (
        id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        name VARCHAR(100) NOT NULL,
        exchange VARCHAR(50) NOT NULL,
        type VARCHAR(20) NOT NULL CHECK (type IN ('private', 'public')),
        api_key TEXT,
        api_secret TEXT,
        status VARCHAR(20) DEFAULT 'active',
        info JSONB DEFAULT '{}',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        deleted_at TIMESTAMP WITH TIME ZONE,
        
        -- Constraints
        CONSTRAINT exchange_bindings_exchange_valid CHECK (exchange IN ('binance', 'kraken', 'gate', 'coinbase', 'virtual')),
        CONSTRAINT exchange_bindings_private_requires_user CHECK (type = 'public' OR user_id IS NOT NULL),
        CONSTRAINT exchange_bindings_private_requires_keys CHECK (
            type = 'public' OR (api_key IS NOT NULL AND api_secret IS NOT NULL)
        )
    );
    
    -- Step 3: Create indexes for exchange_bindings table
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_user_id ON exchange_bindings(user_id);
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_exchange ON exchange_bindings(exchange);
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_type ON exchange_bindings(type);
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_status ON exchange_bindings(status);
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_info ON exchange_bindings USING GIN(info);
    CREATE INDEX IF NOT EXISTS idx_exchange_bindings_deleted_at ON exchange_bindings(deleted_at);
    
    -- Create unique constraint on name per user (allowing NULL user_id for public bindings)
    CREATE UNIQUE INDEX IF NOT EXISTS exchange_bindings_name_unique 
    ON exchange_bindings (COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), name);
    
    -- Step 4: Migrate existing trading data to exchange bindings
    -- Create exchange bindings from existing tradings with API credentials
    INSERT INTO exchange_bindings (user_id, name, exchange, type, api_key, api_secret, status, info, created_at, updated_at)
    SELECT 
        user_id,
        name || ' - Exchange Binding' as name,
        CASE 
            WHEN api_key LIKE '%demo%' OR api_key LIKE '%test%' THEN 'virtual'
            ELSE 'binance'  -- Default to binance, can be updated manually
        END as exchange,
        'private' as type,
        api_key,
        api_secret,
        status,
        '{}'::jsonb as info,
        created_at,
        updated_at
    FROM tradings 
    WHERE api_key IS NOT NULL AND api_secret IS NOT NULL;
    
    -- Step 5: Add exchange_binding_id column to tradings table
    ALTER TABLE tradings ADD COLUMN IF NOT EXISTS exchange_binding_id UUID;
    
    -- Step 6: Update tradings to reference the created exchange bindings
    UPDATE tradings SET exchange_binding_id = eb.id
    FROM exchange_bindings eb
    WHERE tradings.user_id = eb.user_id 
    AND tradings.api_key = eb.api_key 
    AND tradings.api_secret = eb.api_secret;
    
    -- Step 7: For tradings without API credentials, create virtual exchange bindings
    INSERT INTO exchange_bindings (user_id, name, exchange, type, status, info, created_at, updated_at)
    SELECT DISTINCT
        user_id,
        'Virtual Trading Binding' as name,
        'virtual' as exchange,
        'public' as type,
        'active' as status,
        '{}'::jsonb as info,
        NOW(),
        NOW()
    FROM tradings t
    WHERE exchange_binding_id IS NULL
    AND NOT EXISTS (
        SELECT 1 FROM exchange_bindings eb 
        WHERE eb.user_id = t.user_id AND eb.exchange = 'virtual' AND eb.type = 'public'
    );
    
    -- Update tradings without exchange binding to use virtual bindings
    UPDATE tradings SET exchange_binding_id = eb.id
    FROM exchange_bindings eb
    WHERE tradings.exchange_binding_id IS NULL
    AND tradings.user_id = eb.user_id 
    AND eb.exchange = 'virtual' 
    AND eb.type = 'public';
    
    -- Step 8: Make exchange_binding_id NOT NULL and add foreign key
    ALTER TABLE tradings ALTER COLUMN exchange_binding_id SET NOT NULL;
    ALTER TABLE tradings ADD CONSTRAINT tradings_exchange_binding_id_fkey 
        FOREIGN KEY (exchange_binding_id) REFERENCES exchange_bindings(id) ON DELETE RESTRICT;
    
    -- Step 9: Create index on exchange_binding_id
    CREATE INDEX IF NOT EXISTS idx_tradings_exchange_binding_id ON tradings(exchange_binding_id);
    
    -- Step 10: Remove old API credential columns from tradings
    ALTER TABLE tradings DROP COLUMN IF EXISTS api_key;
    ALTER TABLE tradings DROP COLUMN IF EXISTS api_secret;
    
    -- Step 11: Drop old unique constraints on API keys
    DROP INDEX IF EXISTS tradings_user_api_key_active_unique;
    DROP INDEX IF EXISTS tradings_user_api_secret_active_unique;
    
    -- Step 12: Update existing unique constraints
    DROP INDEX IF EXISTS tradings_user_name_active_unique;
    CREATE UNIQUE INDEX IF NOT EXISTS tradings_user_name_active_unique 
    ON tradings (user_id, name) 
    WHERE deleted_at IS NULL;
    
    RAISE NOTICE 'Successfully migrated to exchange bindings architecture';
    
END
$$;

-- Create trigger for exchange_bindings table
CREATE OR REPLACE TRIGGER update_exchange_bindings_updated_at 
    BEFORE UPDATE ON exchange_bindings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();