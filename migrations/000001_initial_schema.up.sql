-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
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
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_info ON users USING GIN(info);

-- OAuth tokens table
CREATE TABLE oauth_tokens (
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
CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_provider ON oauth_tokens(provider);
CREATE UNIQUE INDEX idx_oauth_tokens_provider_user ON oauth_tokens(provider, provider_user_id);
CREATE INDEX idx_oauth_tokens_info ON oauth_tokens USING GIN(info);

-- Exchanges table
CREATE TABLE exchanges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    api_key TEXT NOT NULL,
    api_secret TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for exchanges table
CREATE INDEX idx_exchanges_user_id ON exchanges(user_id);
CREATE INDEX idx_exchanges_type ON exchanges(type);
CREATE INDEX idx_exchanges_status ON exchanges(status);
CREATE INDEX idx_exchanges_info ON exchanges USING GIN(info);

-- Sub-accounts table
CREATE TABLE sub_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_id UUID NOT NULL REFERENCES exchanges(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    balance DECIMAL(20,8) DEFAULT 0,
    info JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for sub_accounts table
CREATE INDEX idx_sub_accounts_user_id ON sub_accounts(user_id);
CREATE INDEX idx_sub_accounts_exchange_id ON sub_accounts(exchange_id);
CREATE INDEX idx_sub_accounts_symbol ON sub_accounts(symbol);
CREATE INDEX idx_sub_accounts_info ON sub_accounts USING GIN(info);

-- Transactions table (time-series data)
CREATE TABLE transactions (
    id UUID DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    exchange_id UUID NOT NULL REFERENCES exchanges(id),
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

-- Convert transactions to hypertable for TimescaleDB
SELECT create_hypertable('transactions', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Create indexes for transactions table
CREATE INDEX idx_transactions_user_id_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX idx_transactions_sub_account_id_timestamp ON transactions(sub_account_id, timestamp DESC);
CREATE INDEX idx_transactions_exchange_id_timestamp ON transactions(exchange_id, timestamp DESC);
CREATE INDEX idx_transactions_direction ON transactions(direction);
CREATE INDEX idx_transactions_reason ON transactions(reason);
CREATE INDEX idx_transactions_info ON transactions USING GIN(info);

-- Trading logs table (time-series data)
CREATE TABLE trading_logs (
    id UUID DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    exchange_id UUID NOT NULL REFERENCES exchanges(id),
    sub_account_id UUID REFERENCES sub_accounts(id),
    transaction_id UUID,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    type VARCHAR(50) NOT NULL,
    source VARCHAR(20) NOT NULL CHECK (source IN ('manual', 'bot')),
    message TEXT NOT NULL,
    info JSONB DEFAULT '{}',
    PRIMARY KEY (id, timestamp)
);

-- Convert trading_logs to hypertable for TimescaleDB
SELECT create_hypertable('trading_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Create indexes for trading_logs table
CREATE INDEX idx_trading_logs_user_id_timestamp ON trading_logs(user_id, timestamp DESC);
CREATE INDEX idx_trading_logs_sub_account_id_timestamp ON trading_logs(sub_account_id, timestamp DESC);
CREATE INDEX idx_trading_logs_exchange_id_timestamp ON trading_logs(exchange_id, timestamp DESC);
CREATE INDEX idx_trading_logs_transaction_id ON trading_logs(transaction_id);
CREATE INDEX idx_trading_logs_type ON trading_logs(type);
CREATE INDEX idx_trading_logs_source ON trading_logs(source);
CREATE INDEX idx_trading_logs_info ON trading_logs USING GIN(info);

-- Event processing table for NATS message deduplication
CREATE TABLE event_processing (
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
CREATE INDEX idx_event_processing_event_id ON event_processing(event_id);
CREATE INDEX idx_event_processing_event_type ON event_processing(event_type);
CREATE INDEX idx_event_processing_user_id ON event_processing(user_id);
CREATE INDEX idx_event_processing_status ON event_processing(status);
CREATE INDEX idx_event_processing_processed_at ON event_processing(processed_at);
CREATE INDEX idx_event_processing_info ON event_processing USING GIN(info);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at columns
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_oauth_tokens_updated_at BEFORE UPDATE ON oauth_tokens 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_exchanges_updated_at BEFORE UPDATE ON exchanges 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sub_accounts_updated_at BEFORE UPDATE ON sub_accounts 
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
    v_exchange_id UUID;
    v_transaction_id UUID;
BEGIN
    -- Get user_id and exchange_id from sub_account
    SELECT user_id, exchange_id INTO v_user_id, v_exchange_id
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
        user_id, exchange_id, sub_account_id, direction, reason, 
        amount, closing_balance, info
    ) VALUES (
        v_user_id, v_exchange_id, p_sub_account_id, p_direction, p_reason,
        p_amount, p_new_balance, p_info
    ) RETURNING id INTO v_transaction_id;
    
    RETURN v_transaction_id;
END;
$$ LANGUAGE plpgsql;