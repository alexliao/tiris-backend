-- Enable TimescaleDB extension (optional for testing)
-- Note: This will fail in test environments without TimescaleDB, which is expected
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'TimescaleDB extension not available - hypertables will be skipped';
END
$$;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    avatar TEXT,
    settings JSONB DEFAULT '{}',
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_info ON users USING GIN(info);

-- OAuth tokens table
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for oauth_tokens table
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user_id ON oauth_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_provider ON oauth_tokens(provider);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_tokens_provider_user ON oauth_tokens(provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_info ON oauth_tokens USING GIN(info);

-- Exchange bindings table
CREATE TABLE IF NOT EXISTS exchange_bindings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    name VARCHAR(100) NOT NULL,
    exchange VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('private', 'public')),
    api_key TEXT,
    api_secret TEXT,
    status VARCHAR(20) DEFAULT 'active',
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT exchange_bindings_exchange_valid CHECK (exchange IN ('binance', 'kraken', 'gate', 'coinbase', 'virtual')),
    CONSTRAINT exchange_bindings_private_requires_user CHECK (type = 'public' OR user_id IS NOT NULL),
    CONSTRAINT exchange_bindings_private_requires_keys CHECK (
        type = 'public' OR (api_key IS NOT NULL AND api_secret IS NOT NULL)
    )
);

-- Create indexes for exchange_bindings table
CREATE INDEX IF NOT EXISTS idx_exchange_bindings_user_id ON exchange_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_exchange_bindings_exchange ON exchange_bindings(exchange);
CREATE INDEX IF NOT EXISTS idx_exchange_bindings_type ON exchange_bindings(type);
CREATE INDEX IF NOT EXISTS idx_exchange_bindings_status ON exchange_bindings(status);
CREATE INDEX IF NOT EXISTS idx_exchange_bindings_info ON exchange_bindings USING GIN(info);

-- Create unique constraint on name per user (allowing NULL user_id for public bindings)
CREATE UNIQUE INDEX IF NOT EXISTS exchange_bindings_name_unique 
ON exchange_bindings (COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), name);

-- Add foreign key constraint for user_id (allows NULL values for public bindings)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_exchange_bindings_user_id' 
        AND table_name = 'exchange_bindings'
    ) THEN
        ALTER TABLE exchange_bindings 
        ADD CONSTRAINT fk_exchange_bindings_user_id 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Tradings table
CREATE TABLE IF NOT EXISTS tradings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_binding_id UUID NOT NULL REFERENCES exchange_bindings(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT tradings_type_valid CHECK (type IN ('real', 'virtual', 'backtest')),
    CONSTRAINT tradings_user_name_unique UNIQUE (user_id, name)
);

-- Create indexes for tradings table
CREATE INDEX IF NOT EXISTS idx_tradings_user_id ON tradings(user_id);
CREATE INDEX IF NOT EXISTS idx_tradings_exchange_binding_id ON tradings(exchange_binding_id);
CREATE INDEX IF NOT EXISTS idx_tradings_type ON tradings(type);
CREATE INDEX IF NOT EXISTS idx_tradings_status ON tradings(status);
CREATE INDEX IF NOT EXISTS idx_tradings_info ON tradings USING GIN(info);

-- Sub-accounts table
CREATE TABLE IF NOT EXISTS sub_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trading_id UUID NOT NULL REFERENCES tradings(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    balance DECIMAL(20,8) DEFAULT 0,
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for sub_accounts table
CREATE INDEX IF NOT EXISTS idx_sub_accounts_user_id ON sub_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_sub_accounts_trading_id ON sub_accounts(trading_id);
CREATE INDEX IF NOT EXISTS idx_sub_accounts_symbol ON sub_accounts(symbol);
CREATE INDEX IF NOT EXISTS idx_sub_accounts_info ON sub_accounts USING GIN(info);

-- Transactions table (time-series data)
CREATE TABLE IF NOT EXISTS transactions (
    id UUID DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_id UUID NOT NULL REFERENCES tradings(id),
    sub_account_id UUID NOT NULL REFERENCES sub_accounts(id),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('debit', 'credit')),
    reason VARCHAR(50) NOT NULL,
    amount DECIMAL(20,8) NOT NULL,
    closing_balance DECIMAL(20,8) NOT NULL,
    price DECIMAL(20,8),
    quote_symbol VARCHAR(20),
    info JSONB DEFAULT '{}',
    PRIMARY KEY (id, timestamp)
);

-- Convert transactions to hypertable for TimescaleDB (conditional)
DO $$
BEGIN
    PERFORM create_hypertable('transactions', 'timestamp', chunk_time_interval => INTERVAL '1 day');
    RAISE NOTICE 'Created hypertable for transactions';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'TimescaleDB not available - transactions table will be regular table';
END
$$;

-- Create indexes for transactions table
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_sub_account_id_timestamp ON transactions(sub_account_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_trading_id_timestamp ON transactions(trading_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_direction ON transactions(direction);
CREATE INDEX IF NOT EXISTS idx_transactions_reason ON transactions(reason);
CREATE INDEX IF NOT EXISTS idx_transactions_info ON transactions USING GIN(info);

-- Trading logs table (time-series data)
CREATE TABLE IF NOT EXISTS trading_logs (
    id UUID DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    trading_id UUID NOT NULL REFERENCES tradings(id),
    sub_account_id UUID REFERENCES sub_accounts(id),
    transaction_id UUID,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    type VARCHAR(50) NOT NULL,
    source VARCHAR(20) NOT NULL CHECK (source IN ('manual', 'bot')),
    message TEXT NOT NULL,
    info JSONB DEFAULT '{}',
    PRIMARY KEY (id, timestamp)
);

-- Convert trading_logs to hypertable for TimescaleDB (conditional)
DO $$
BEGIN
    PERFORM create_hypertable('trading_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');
    RAISE NOTICE 'Created hypertable for trading_logs';
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'TimescaleDB not available - trading_logs table will be regular table';
END
$$;

-- Create indexes for trading_logs table
CREATE INDEX IF NOT EXISTS idx_trading_logs_user_id_timestamp ON trading_logs(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trading_logs_sub_account_id_timestamp ON trading_logs(sub_account_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trading_logs_trading_id_timestamp ON trading_logs(trading_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_trading_logs_transaction_id ON trading_logs(transaction_id);
CREATE INDEX IF NOT EXISTS idx_trading_logs_type ON trading_logs(type);
CREATE INDEX IF NOT EXISTS idx_trading_logs_source ON trading_logs(source);
CREATE INDEX IF NOT EXISTS idx_trading_logs_info ON trading_logs USING GIN(info);

-- Event processing table for NATS message deduplication
CREATE TABLE IF NOT EXISTS event_processing (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    user_id UUID REFERENCES users(id),
    sub_account_id UUID REFERENCES sub_accounts(id),
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) DEFAULT 'processed',
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    info JSONB DEFAULT '{}'
);

-- Create indexes for event_processing table
CREATE INDEX IF NOT EXISTS idx_event_processing_event_id ON event_processing(event_id);
CREATE INDEX IF NOT EXISTS idx_event_processing_event_type ON event_processing(event_type);
CREATE INDEX IF NOT EXISTS idx_event_processing_user_id ON event_processing(user_id);
CREATE INDEX IF NOT EXISTS idx_event_processing_status ON event_processing(status);
CREATE INDEX IF NOT EXISTS idx_event_processing_processed_at ON event_processing(processed_at);
CREATE INDEX IF NOT EXISTS idx_event_processing_info ON event_processing USING GIN(info);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at columns
CREATE OR REPLACE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON oauth_tokens 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_exchange_bindings_updated_at BEFORE UPDATE ON exchange_bindings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_tradings_updated_at BEFORE UPDATE ON tradings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_sub_accounts_updated_at BEFORE UPDATE ON sub_accounts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update sub-account balance and create transaction record
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
$$ LANGUAGE plpgsql;

-- Insert public exchange bindings for simulation and backtesting
INSERT INTO exchange_bindings (user_id, name, exchange, type, api_key, api_secret, status, info) VALUES
(NULL, 'Binance', 'binance', 'public', NULL, NULL, 'active', '{"description": "A virtual Binance exchange for simulation and backtesting"}'),
(NULL, 'Kraken', 'kraken', 'public', NULL, NULL, 'active', '{"description": "A virtual Kraken exchange for simulation and backtesting"}'),
(NULL, 'Gate.io', 'gate', 'public', NULL, NULL, 'active', '{"description": "A virtual Gate.io exchange for simulation and backtesting"}'),
(NULL, 'Coinbase', 'coinbase', 'public', NULL, NULL, 'active', '{"description": "A virtual Coinbase exchange for simulation and backtesting"}'),
(NULL, 'Virtual', 'virtual', 'public', NULL, NULL, 'active', '{"description": "A completely virtual exchange for testing and simulation"}')
ON CONFLICT (COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), name) DO NOTHING;