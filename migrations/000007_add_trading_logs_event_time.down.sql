-- Remove event_time column and related indexes from trading_logs table

-- Drop indexes first
DROP INDEX IF EXISTS idx_trading_logs_event_time;
DROP INDEX IF EXISTS idx_trading_logs_user_event_time;
DROP INDEX IF EXISTS idx_trading_logs_trading_event_time;

-- Remove the event_time column
ALTER TABLE trading_logs DROP COLUMN IF EXISTS event_time;

-- Remove the updated comment on timestamp column
COMMENT ON COLUMN trading_logs.timestamp IS NULL;