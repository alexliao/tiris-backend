# Tiris Backend Database Schema Design

## 1. Database Overview

### 1.1 Database Technology
- **Primary Database**: PostgreSQL 15+
- **Extension**: TimescaleDB for time-series data
- **Schema Name**: `tiris`
- **Character Set**: UTF-8
- **Timezone**: UTC

### 1.2 Design Principles
- ACID compliance for all financial operations
- Time-series optimization for trading data
- Normalized design with selective denormalization for performance
- Comprehensive audit trails
- Flexible JSON storage for extensibility

## 2. Core Tables

### 2.1 Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    avatar TEXT,
    settings JSONB DEFAULT '{}',
    info JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deleted')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT users_username_format CHECK (username ~ '^[a-zA-Z0-9_]{3,50}$'),
    CONSTRAINT users_email_format CHECK (email ~ '^[^@]+@[^@]+\.[^@]+$')
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_settings_gin ON users USING gin(settings);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `username`: User's display name (3-50 characters, alphanumeric + underscore)
- `email`: User's email address (unique)
- `avatar`: URL to user's avatar image
- `settings`: User preferences (timezone, currency, notifications, etc.)
- `info`: Extended user information (profile data, OAuth details)
- `status`: Account status (active, disabled, deleted)

### 2.2 Exchange Bindings Table

```sql
CREATE TABLE exchange_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    exchange VARCHAR(50) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('private', 'public')),
    api_key_encrypted TEXT,
    api_secret_encrypted TEXT,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error')),
    info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT exchange_bindings_exchange_valid CHECK (exchange IN ('binance', 'kraken', 'gate', 'coinbase', 'virtual')),
    CONSTRAINT exchange_bindings_name_unique UNIQUE (COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), name),
    CONSTRAINT exchange_bindings_private_requires_user CHECK (type = 'public' OR user_id IS NOT NULL),
    CONSTRAINT exchange_bindings_private_requires_keys CHECK (
        type = 'public' OR (api_key_encrypted IS NOT NULL AND api_secret_encrypted IS NOT NULL)
    )
);

-- Indexes
CREATE INDEX idx_exchange_bindings_user_id ON exchange_bindings(user_id);
CREATE INDEX idx_exchange_bindings_exchange ON exchange_bindings(exchange);
CREATE INDEX idx_exchange_bindings_type ON exchange_bindings(type);
CREATE INDEX idx_exchange_bindings_status ON exchange_bindings(status);
CREATE INDEX idx_exchange_bindings_info_gin ON exchange_bindings USING gin(info);

-- Trigger for updated_at
CREATE TRIGGER update_exchange_bindings_updated_at 
    BEFORE UPDATE ON exchange_bindings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `user_id`: Foreign key to users table (NULL for public bindings)
- `name`: User-defined name for the exchange binding
- `exchange`: Exchange name (binance, kraken, gate, coinbase, virtual)
- `type`: Binding type (private for user credentials, public for system-wide)
- `api_key_encrypted`: Encrypted API key (NULL for public bindings)
- `api_secret_encrypted`: Encrypted API secret (NULL for public bindings)
- `status`: Exchange connection status
- `info`: Additional exchange data (permissions, testnet flag, description)

### 2.3 Tradings Table

```sql
CREATE TABLE tradings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange_binding_id UUID NOT NULL REFERENCES exchange_bindings(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'paused')),
    info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT tradings_type_valid CHECK (type IN ('real', 'virtual', 'backtest')),
    CONSTRAINT tradings_user_name_unique UNIQUE (user_id, name)
);

-- Indexes
CREATE INDEX idx_tradings_user_id ON tradings(user_id);
CREATE INDEX idx_tradings_exchange_binding_id ON tradings(exchange_binding_id);
CREATE INDEX idx_tradings_type ON tradings(type);
CREATE INDEX idx_tradings_status ON tradings(status);
CREATE INDEX idx_tradings_info_gin ON tradings USING gin(info);

-- Trigger for updated_at
CREATE TRIGGER update_tradings_updated_at 
    BEFORE UPDATE ON tradings 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `user_id`: Foreign key to users table
- `exchange_binding_id`: Foreign key to exchange_bindings table
- `name`: User-defined name for the trading
- `type`: Trading type (real, virtual, backtest)
- `status`: Trading status (active, inactive, paused)
- `info`: Additional trading data (strategies, settings, performance metrics)

### 2.4 Sub-accounts Table

```sql
CREATE TABLE sub_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trading_id UUID NOT NULL REFERENCES tradings(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    balance DECIMAL(20,8) DEFAULT 0 CHECK (balance >= 0),
    info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT sub_accounts_trading_name_unique UNIQUE (trading_id, name)
);

-- Indexes
CREATE INDEX idx_sub_accounts_user_id ON sub_accounts(user_id);
CREATE INDEX idx_sub_accounts_trading_id ON sub_accounts(trading_id);
CREATE INDEX idx_sub_accounts_symbol ON sub_accounts(symbol);
CREATE INDEX idx_sub_accounts_balance ON sub_accounts(balance);

-- Trigger for updated_at
CREATE TRIGGER update_sub_accounts_updated_at 
    BEFORE UPDATE ON sub_accounts 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `user_id`: Foreign key to users table
- `trading_id`: Foreign key to tradings table
- `name`: User-defined sub-account name
- `symbol`: Asset symbol (BTC, ETH, USDT, etc.)
- `balance`: Current available balance (updated when orders complete)
- `info`: Additional sub-account data (initial balance, trading rules, bot configuration)

### 2.5 Transactions Table (Time-series)

```sql
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trading_id UUID NOT NULL REFERENCES tradings(id) ON DELETE CASCADE,
    sub_account_id UUID NOT NULL REFERENCES sub_accounts(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('debit', 'credit')),
    reason VARCHAR(50) NOT NULL,
    amount DECIMAL(20,8) NOT NULL CHECK (amount != 0),
    closing_balance DECIMAL(20,8) NOT NULL CHECK (closing_balance >= 0),
    price DECIMAL(20,8),
    quote_symbol VARCHAR(20),
    info JSONB DEFAULT '{}',
    
    -- Constraints
    CONSTRAINT transactions_reason_valid CHECK (
        reason IN ('long', 'short', 'stop_loss', 'take_profit', 'dividend', 
                  'transfer', 'fee', 'deposit', 'withdrawal', 'interest', 'rebate')
    )
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('transactions', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Indexes
CREATE INDEX idx_transactions_user_timestamp ON transactions(user_id, timestamp DESC);
CREATE INDEX idx_transactions_sub_account_timestamp ON transactions(sub_account_id, timestamp DESC);
CREATE INDEX idx_transactions_trading_timestamp ON transactions(trading_id, timestamp DESC);
CREATE INDEX idx_transactions_reason ON transactions(reason);
CREATE INDEX idx_transactions_direction ON transactions(direction);
CREATE INDEX idx_transactions_info_gin ON transactions USING gin(info);

-- Compression policy (compress chunks older than 7 days)
SELECT add_compression_policy('transactions', INTERVAL '7 days');

-- Retention policy (drop chunks older than 2 years)
SELECT add_retention_policy('transactions', INTERVAL '2 years');
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `user_id`: Foreign key to users table
- `trading_id`: Foreign key to tradings table
- `sub_account_id`: Foreign key to sub_accounts table
- `timestamp`: Transaction timestamp (indexed for time-series queries)
- `direction`: Transaction direction (debit/credit)
- `reason`: Transaction reason/type
- `amount`: Transaction amount (positive for credit, negative for debit)
- `closing_balance`: Balance after transaction
- `price`: Asset price at transaction time
- `quote_symbol`: Quote currency for price
- `info`: Additional transaction data (order details, fees, etc.)

### 2.6 Trading Logs Table (Time-series)

```sql
CREATE TABLE trading_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trading_id UUID NOT NULL REFERENCES tradings(id) ON DELETE CASCADE,
    sub_account_id UUID REFERENCES sub_accounts(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_time TIMESTAMPTZ,
    type VARCHAR(50) NOT NULL,
    source VARCHAR(20) NOT NULL CHECK (source IN ('manual', 'bot')),
    message TEXT,
    info JSONB DEFAULT '{}',
    
    -- Constraints
    CONSTRAINT trading_logs_type_valid CHECK (
        type IN ('buy_order', 'sell_order', 'prediction', 'chart_analysis', 
                'strategy_signal', 'risk_management', 'portfolio_rebalance',
                'market_data', 'system_alert', 'error')
    )
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('trading_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Indexes
CREATE INDEX idx_trading_logs_user_timestamp ON trading_logs(user_id, timestamp DESC);
CREATE INDEX idx_trading_logs_trading_timestamp ON trading_logs(trading_id, timestamp DESC);
CREATE INDEX idx_trading_logs_sub_account_timestamp ON trading_logs(sub_account_id, timestamp DESC);
CREATE INDEX idx_trading_logs_event_time ON trading_logs(event_time DESC) WHERE event_time IS NOT NULL;
CREATE INDEX idx_trading_logs_user_event_time ON trading_logs(user_id, event_time DESC) WHERE event_time IS NOT NULL;
CREATE INDEX idx_trading_logs_trading_event_time ON trading_logs(trading_id, event_time DESC) WHERE event_time IS NOT NULL;
CREATE INDEX idx_trading_logs_type ON trading_logs(type);
CREATE INDEX idx_trading_logs_source ON trading_logs(source);
CREATE INDEX idx_trading_logs_transaction_id ON trading_logs(transaction_id);
CREATE INDEX idx_trading_logs_info_gin ON trading_logs USING gin(info);

-- Compression policy (compress chunks older than 30 days)
SELECT add_compression_policy('trading_logs', INTERVAL '30 days');

-- Retention policy (drop chunks older than 1 year)
SELECT add_retention_policy('trading_logs', INTERVAL '1 year');
```

**Fields Description:**
- `id`: Unique identifier (UUID)
- `user_id`: Foreign key to users table
- `trading_id`: Foreign key to tradings table
- `sub_account_id`: Optional foreign key to sub_accounts table
- `transaction_id`: Optional foreign key to related transaction
- `timestamp`: Log creation timestamp (when record was inserted into database)
- `event_time`: Logical event timestamp (when trading event actually occurred, NULL for backward compatibility)
- `type`: Log type/category
- `source`: Log source (manual user action or bot)
- `message`: Human-readable log message
- `info`: Additional log data (order details, predictions, charts)

## 3. Supporting Tables

### 3.1 OAuth Tokens Table

```sql
CREATE TABLE oauth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    access_token_encrypted TEXT,
    refresh_token_encrypted TEXT,
    expires_at TIMESTAMPTZ,
    scope TEXT,
    info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT oauth_tokens_provider_valid CHECK (provider IN ('google', 'wechat')),
    CONSTRAINT oauth_tokens_user_provider_unique UNIQUE (user_id, provider)
);

-- Indexes
CREATE INDEX idx_oauth_tokens_user_id ON oauth_tokens(user_id);
CREATE INDEX idx_oauth_tokens_provider ON oauth_tokens(provider);
CREATE INDEX idx_oauth_tokens_expires_at ON oauth_tokens(expires_at);

-- Trigger for updated_at
CREATE TRIGGER update_oauth_tokens_updated_at 
    BEFORE UPDATE ON oauth_tokens 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
```

### 3.2 Event Processing Table

```sql
CREATE TABLE event_processing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    trading_id UUID REFERENCES tradings(id) ON DELETE SET NULL,
    sub_account_id UUID REFERENCES sub_accounts(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'retrying')),
    attempt_count INTEGER DEFAULT 1,
    event_data JSONB NOT NULL,
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    info JSONB DEFAULT '{}',
    
    -- Constraints
    CONSTRAINT event_processing_attempt_count_positive CHECK (attempt_count > 0)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('event_processing', 'created_at', chunk_time_interval => INTERVAL '1 day');

-- Indexes
CREATE INDEX idx_event_processing_status ON event_processing(status);
CREATE INDEX idx_event_processing_event_type ON event_processing(event_type);
CREATE INDEX idx_event_processing_user_created ON event_processing(user_id, created_at DESC);
CREATE INDEX idx_event_processing_event_id ON event_processing(event_id);
CREATE INDEX idx_event_processing_subject ON event_processing(subject);

-- Trigger for updated_at
CREATE TRIGGER update_event_processing_updated_at 
    BEFORE UPDATE ON event_processing 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Compression policy (compress chunks older than 7 days)
SELECT add_compression_policy('event_processing', INTERVAL '7 days');

-- Retention policy (drop chunks older than 6 months)
SELECT add_retention_policy('event_processing', INTERVAL '6 months');
```

### 3.3 Audit Logs Table

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    info JSONB DEFAULT '{}'
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('audit_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day');

-- Indexes
CREATE INDEX idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);

-- Retention policy (drop chunks older than 3 years)
SELECT add_retention_policy('audit_logs', INTERVAL '3 years');
```

## 4. Views and Materialized Views

### 4.1 User Portfolio View

```sql
CREATE VIEW user_portfolios AS
SELECT 
    u.id as user_id,
    u.username,
    t.id as trading_id,
    t.name as trading_name,
    t.type as trading_type,
    eb.exchange as exchange_name,
    eb.type as exchange_type,
    COUNT(sa.id) as sub_account_count,
    COALESCE(SUM(CASE WHEN sa.symbol = 'BTC' THEN sa.balance ELSE 0 END), 0) as btc_balance,
    COALESCE(SUM(CASE WHEN sa.symbol = 'ETH' THEN sa.balance ELSE 0 END), 0) as eth_balance,
    COALESCE(SUM(CASE WHEN sa.symbol = 'USDT' THEN sa.balance ELSE 0 END), 0) as usdt_balance,
    t.created_at as trading_added_at
FROM users u
LEFT JOIN tradings t ON u.id = t.user_id AND t.status = 'active'
LEFT JOIN exchange_bindings eb ON t.exchange_binding_id = eb.id AND eb.status = 'active'
LEFT JOIN sub_accounts sa ON t.id = sa.trading_id
WHERE u.status = 'active'
GROUP BY u.id, u.username, t.id, t.name, t.type, eb.exchange, eb.type, t.created_at;
```

### 4.2 Daily Transaction Summary (Materialized View)

```sql
CREATE MATERIALIZED VIEW daily_transaction_summary AS
SELECT 
    user_id,
    trading_id,
    sub_account_id,
    DATE_TRUNC('day', timestamp) as date,
    COUNT(*) as transaction_count,
    SUM(CASE WHEN direction = 'credit' THEN amount ELSE 0 END) as total_credits,
    SUM(CASE WHEN direction = 'debit' THEN ABS(amount) ELSE 0 END) as total_debits,
    AVG(closing_balance) as avg_balance,
    MAX(closing_balance) as max_balance,
    MIN(closing_balance) as min_balance,
    COUNT(DISTINCT reason) as unique_reasons
FROM transactions
GROUP BY user_id, trading_id, sub_account_id, DATE_TRUNC('day', timestamp);

-- Indexes
CREATE INDEX idx_daily_transaction_summary_date ON daily_transaction_summary(date DESC);
CREATE INDEX idx_daily_transaction_summary_user ON daily_transaction_summary(user_id, date DESC);

-- Refresh policy (refresh every hour)
CREATE OR REPLACE FUNCTION refresh_daily_transaction_summary()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY daily_transaction_summary;
END;
$$ LANGUAGE plpgsql;

-- Schedule refresh (requires pg_cron extension)
-- SELECT cron.schedule('refresh-daily-summary', '0 * * * *', 'SELECT refresh_daily_transaction_summary();');
```

## 5. Functions and Stored Procedures

### 5.1 Account Balance Update Function

```sql
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
    v_transaction_id UUID;
    v_user_id UUID;
    v_trading_id UUID;
BEGIN
    -- Get related IDs
    SELECT user_id, trading_id 
    INTO v_user_id, v_trading_id
    FROM sub_accounts 
    WHERE id = p_sub_account_id;
    
    -- Validate new balance is non-negative
    IF p_new_balance < 0 THEN
        RAISE EXCEPTION 'Invalid balance from bot: %', p_new_balance;
    END IF;
    
    -- Update sub-account balance to the value provided by bot
    UPDATE sub_accounts 
    SET balance = p_new_balance, updated_at = NOW()
    WHERE id = p_sub_account_id;
    
    -- Create transaction record
    INSERT INTO transactions (
        user_id, trading_id, sub_account_id,
        direction, reason, amount, closing_balance, info
    ) VALUES (
        v_user_id, v_trading_id, p_sub_account_id,
        p_direction, p_reason, p_amount, p_new_balance, p_info
    ) RETURNING id INTO v_transaction_id;
    
    RETURN v_transaction_id;
END;
$$ LANGUAGE plpgsql;
```

### 5.2 Event Processing Function

```sql
CREATE OR REPLACE FUNCTION process_trading_event(
    p_event_id VARCHAR(255),
    p_event_type VARCHAR(100),
    p_subject VARCHAR(255),
    p_event_data JSONB
)
RETURNS UUID AS $$
DECLARE
    v_processing_id UUID;
    v_user_id UUID;
    v_trading_id UUID;
    v_sub_account_id UUID;
BEGIN
    -- Extract common fields from event data
    v_user_id := (p_event_data->>'user_id')::UUID;
    v_trading_id := (p_event_data->>'trading_id')::UUID;
    v_sub_account_id := (p_event_data->>'sub_account_id')::UUID;
    
    -- Check for duplicate event
    SELECT id INTO v_processing_id 
    FROM event_processing 
    WHERE event_id = p_event_id;
    
    IF v_processing_id IS NOT NULL THEN
        -- Event already processed or in progress
        RETURN v_processing_id;
    END IF;
    
    -- Insert event processing record
    INSERT INTO event_processing (
        event_id, event_type, subject, user_id, trading_id, sub_account_id,
        status, event_data
    ) VALUES (
        p_event_id, p_event_type, p_subject, v_user_id, v_trading_id, v_sub_account_id,
        'pending', p_event_data
    ) RETURNING id INTO v_processing_id;
    
    RETURN v_processing_id;
END;
$$ LANGUAGE plpgsql;
```

### 5.3 User Statistics Function

```sql
CREATE OR REPLACE FUNCTION get_user_statistics(p_user_id UUID)
RETURNS TABLE(
    total_tradings INT,
    active_tradings INT,
    total_sub_accounts INT,
    total_transactions BIGINT,
    first_transaction_date TIMESTAMPTZ,
    last_transaction_date TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(DISTINCT tr.id)::INT as total_tradings,
        COUNT(DISTINCT CASE WHEN tr.status = 'active' THEN tr.id END)::INT as active_tradings,
        COUNT(DISTINCT sa.id)::INT as total_sub_accounts,
        COALESCE(COUNT(t.id), 0) as total_transactions,
        MIN(t.timestamp) as first_transaction_date,
        MAX(t.timestamp) as last_transaction_date
    FROM users u
    LEFT JOIN tradings tr ON u.id = tr.user_id
    LEFT JOIN sub_accounts sa ON tr.id = sa.trading_id
    LEFT JOIN transactions t ON sa.id = t.sub_account_id
    WHERE u.id = p_user_id
    GROUP BY u.id;
END;
$$ LANGUAGE plpgsql;
```

## 6. Performance Optimizations

### 6.1 Partitioning Strategy

**Time-based Partitioning:**
- `transactions`: Partitioned by month (TimescaleDB automatic)
- `trading_logs`: Partitioned by month (TimescaleDB automatic)
- `audit_logs`: Partitioned by month (TimescaleDB automatic)

**Benefits:**
- Faster queries with time-based filtering
- Efficient data archival and deletion
- Improved maintenance operations

### 6.2 Indexing Strategy

**Primary Indexes:**
- All primary keys (automatic)
- Foreign keys for join performance
- Time-based indexes for time-series queries

**Composite Indexes:**
- (user_id, timestamp) for user-specific time queries
- (trading_id, timestamp) for trading-specific queries
- (sub_account_id, timestamp) for account-specific queries

**GIN Indexes:**
- JSONB columns for flexible querying
- Full-text search on message fields

### 6.3 Query Optimization Examples

```sql
-- Efficient user portfolio query
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM user_portfolios WHERE user_id = $1;

-- Efficient transaction history query
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM transactions 
WHERE user_id = $1 
  AND timestamp >= $2 
  AND timestamp <= $3
ORDER BY timestamp DESC 
LIMIT 100;

-- Efficient balance calculation
EXPLAIN (ANALYZE, BUFFERS)
SELECT 
    sa.id,
    sa.balance,
    COALESCE(SUM(CASE WHEN t.direction = 'credit' THEN t.amount ELSE -t.amount END), 0) as calculated_balance
FROM sub_accounts sa
LEFT JOIN transactions t ON sa.id = t.sub_account_id
WHERE sa.user_id = $1
GROUP BY sa.id, sa.balance;
```

## 7. Data Integrity and Constraints

### 7.1 Business Rules Enforcement

```sql
-- Ensure sub-account balances are never negative
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_balance_positive 
CHECK (balance >= 0);

-- Ensure locked balance doesn't exceed total balance
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_locked_balance_valid 
CHECK (locked_balance <= balance);

-- Ensure transaction amounts are never zero
ALTER TABLE transactions ADD CONSTRAINT transactions_amount_nonzero 
CHECK (amount != 0);

-- Ensure closing balance is never negative
ALTER TABLE transactions ADD CONSTRAINT transactions_closing_balance_positive 
CHECK (closing_balance >= 0);
```

### 7.2 Uniqueness Constraints

**Global Uniqueness:**
- `users.email`: Emails must be globally unique across all users
- `users.username`: Usernames must be globally unique across all users

**Per-User Uniqueness:**
- `exchange_bindings.name`: Exchange binding names must be unique within each user's account (or globally for public bindings)
- `tradings.name`: Trading names must be unique within each user's account

**Per-Trading Uniqueness:**
- `sub_accounts.name`: Sub-account names must be unique within each trading

```sql
-- Exchange binding uniqueness constraints 
ALTER TABLE exchange_bindings ADD CONSTRAINT exchange_bindings_name_unique 
    UNIQUE (COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), name);

-- Trading uniqueness constraints (per user)
ALTER TABLE tradings ADD CONSTRAINT tradings_user_name_unique UNIQUE (user_id, name);

-- Sub-account uniqueness constraints (per trading)  
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_trading_name_unique UNIQUE (trading_id, name);
```

### 7.3 Referential Integrity

**Foreign Key Policies:**
- `CASCADE`: Delete child records when parent is deleted (users -> exchange_bindings, users -> tradings -> sub_accounts)
- `SET NULL`: Set foreign key to NULL when referenced record is deleted (optional references)
- `RESTRICT`: Prevent deletion if child records exist (exchange_bindings referenced by tradings)

### 7.4 Data Validation

```sql
-- Email format validation
ALTER TABLE users ADD CONSTRAINT users_email_valid 
CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- Username format validation
ALTER TABLE users ADD CONSTRAINT users_username_valid 
CHECK (username ~* '^[A-Za-z0-9_]{3,50}$');

-- Symbol format validation
ALTER TABLE sub_accounts ADD CONSTRAINT sub_accounts_symbol_valid 
CHECK (symbol ~* '^[A-Z]{2,10}$');
```

## 8. Security Considerations

### 8.1 Data Encryption

**Encrypted Fields:**
- `exchange_bindings.api_key_encrypted`
- `exchange_bindings.api_secret_encrypted`
- `oauth_tokens.access_token_encrypted`
- `oauth_tokens.refresh_token_encrypted`

**Encryption Strategy:**
- Use application-level encryption with AES-256
- Store encryption keys in secure key management system
- Rotate encryption keys regularly

### 8.2 Access Control

```sql
-- Create roles for different access levels
CREATE ROLE tiris_read;
CREATE ROLE tiris_write;
CREATE ROLE tiris_admin;

-- Grant appropriate permissions
GRANT SELECT ON ALL TABLES IN SCHEMA public TO tiris_read;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO tiris_write;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tiris_admin;

-- Create application user
CREATE USER tiris_app WITH PASSWORD 'secure_password';
GRANT tiris_write TO tiris_app;
```

### 8.3 Row-Level Security

```sql
-- Enable RLS on sensitive tables
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE exchange_bindings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tradings ENABLE ROW LEVEL SECURITY;
ALTER TABLE sub_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE trading_logs ENABLE ROW LEVEL SECURITY;

-- Create policies for user data access
CREATE POLICY user_own_data ON users
FOR ALL TO tiris_app
USING (id = current_setting('app.current_user_id')::UUID);

CREATE POLICY exchange_binding_own_data ON exchange_bindings
FOR ALL TO tiris_app
USING (user_id = current_setting('app.current_user_id')::UUID OR type = 'public');

CREATE POLICY trading_own_data ON tradings
FOR ALL TO tiris_app
USING (user_id = current_setting('app.current_user_id')::UUID);

-- Similar policies for other tables...
```

## 9. Backup and Recovery

### 9.1 Backup Strategy

**Regular Backups:**
- Full database backup: Daily
- Incremental backup: Every 6 hours
- Transaction log backup: Every 15 minutes

**Backup Retention:**
- Daily backups: 30 days
- Weekly backups: 12 weeks
- Monthly backups: 12 months
- Yearly backups: 7 years

### 9.2 Point-in-Time Recovery

```sql
-- Enable WAL archiving for PITR
archive_mode = on
archive_command = 'cp %p /backup/archive/%f'
wal_level = replica
```

## 10. Monitoring and Maintenance

### 10.1 Health Check Queries

```sql
-- Check for blocking queries
SELECT * FROM pg_stat_activity 
WHERE state = 'active' AND waiting = true;

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    indexrelname,
    idx_tup_read,
    idx_tup_fetch,
    idx_scan
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

### 10.2 Maintenance Tasks

**Automated Maintenance:**
- `VACUUM ANALYZE`: Daily during low-usage hours
- `REINDEX`: Weekly for heavily updated indexes
- Statistics update: Daily
- Connection cleanup: Every hour

**Manual Maintenance:**
- Schema changes and migrations
- Index optimization based on query patterns
- Partition maintenance for time-series tables
- Archive old data based on retention policies