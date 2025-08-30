-- Add event_time column to trading_logs table
-- This column stores the logical time when the trading event actually occurred
-- (e.g., historical time during backtesting vs. current time when creating the record)

-- Add the event_time column
ALTER TABLE trading_logs ADD COLUMN event_time TIMESTAMPTZ;

-- Add comment to document the purpose
COMMENT ON COLUMN trading_logs.event_time IS 'Logical timestamp when the trading event occurred (NULL for backward compatibility). For backtesting, this is the historical time; for live trading, this is when the event actually happened in the market.';

-- Add indexes for efficient time-based queries on event_time
CREATE INDEX idx_trading_logs_event_time ON trading_logs(event_time DESC) WHERE event_time IS NOT NULL;
CREATE INDEX idx_trading_logs_user_event_time ON trading_logs(user_id, event_time DESC) WHERE event_time IS NOT NULL;
CREATE INDEX idx_trading_logs_trading_event_time ON trading_logs(trading_id, event_time DESC) WHERE event_time IS NOT NULL;

-- Update the comment on the existing timestamp column for clarity
COMMENT ON COLUMN trading_logs.timestamp IS 'Database creation timestamp (when record was inserted). This is different from event_time which stores when the trading event logically occurred.';