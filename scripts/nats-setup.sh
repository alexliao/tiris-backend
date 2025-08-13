#!/bin/bash

# NATS JetStream setup script for development environment
# This script creates all necessary streams and consumers for the Tiris Backend

set -e

NATS_SERVER="nats:4222"
MAX_RETRIES=30
RETRY_INTERVAL=2

echo "Setting up NATS JetStream configuration..."

# Wait for NATS server to be available
echo "Waiting for NATS server at $NATS_SERVER..."
for i in $(seq 1 $MAX_RETRIES); do
    if nats --server="$NATS_SERVER" server ping > /dev/null 2>&1; then
        echo "NATS server is ready!"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        echo "Failed to connect to NATS server after $MAX_RETRIES attempts"
        exit 1
    fi
    echo "Attempt $i/$MAX_RETRIES failed, retrying in $RETRY_INTERVAL seconds..."
    sleep $RETRY_INTERVAL
done

# Function to create stream if it doesn't exist
create_stream_if_not_exists() {
    local stream_name=$1
    local subjects=$2
    local retention=$3
    local max_age=$4
    local storage=$5
    
    echo "Checking if stream '$stream_name' exists..."
    if nats --server="$NATS_SERVER" stream info "$stream_name" > /dev/null 2>&1; then
        echo "Stream '$stream_name' already exists, skipping creation"
    else
        echo "Creating stream '$stream_name'..."
        nats --server="$NATS_SERVER" stream add \
            --subjects="$subjects" \
            --retention="$retention" \
            --max-age="$max_age" \
            --storage="$storage" \
            --replicas=1 \
            --discard=old \
            --max-msg-size=1MB \
            --max-msgs=100000 \
            --dupe-window=2m \
            "$stream_name"
        echo "Stream '$stream_name' created successfully"
    fi
}

# Function to create consumer if it doesn't exist
create_consumer_if_not_exists() {
    local stream_name=$1
    local consumer_name=$2
    local filter_subject=$3
    local queue_group=$4
    
    echo "Checking if consumer '$consumer_name' exists in stream '$stream_name'..."
    if nats --server="$NATS_SERVER" consumer info "$stream_name" "$consumer_name" > /dev/null 2>&1; then
        echo "Consumer '$consumer_name' already exists, skipping creation"
    else
        echo "Creating consumer '$consumer_name' in stream '$stream_name'..."
        nats --server="$NATS_SERVER" consumer add \
            --filter="$filter_subject" \
            --ack=explicit \
            --pull \
            --deliver=all \
            --max-deliver=3 \
            --wait=30s \
            --replay=instant \
            ${queue_group:+--queue="$queue_group"} \
            "$stream_name" \
            "$consumer_name"
        echo "Consumer '$consumer_name' created successfully"
    fi
}

# Create Trading Events Stream
echo "Setting up Trading Events stream..."
create_stream_if_not_exists \
    "TRADING_EVENTS" \
    "trading.orders.*,trading.balance.*,trading.signals,trading.errors" \
    "interest" \
    "24h" \
    "file"

# Create System Events Stream  
echo "Setting up System Events stream..."
create_stream_if_not_exists \
    "SYSTEM_EVENTS" \
    "system.*" \
    "interest" \
    "1h" \
    "file"

# Create consumers for order events
echo "Setting up Order Event consumers..."
create_consumer_if_not_exists \
    "TRADING_EVENTS" \
    "order-processor" \
    "trading.orders.*" \
    "order-processors"

# Create consumers for balance events
echo "Setting up Balance Event consumers..."
create_consumer_if_not_exists \
    "TRADING_EVENTS" \
    "balance-processor" \
    "trading.balance.*" \
    "balance-processors"

# Create consumers for error events
echo "Setting up Error Event consumers..."
create_consumer_if_not_exists \
    "TRADING_EVENTS" \
    "error-processor" \
    "trading.errors" \
    "error-processors"

# Create consumers for signal events
echo "Setting up Signal Event consumers..."
create_consumer_if_not_exists \
    "TRADING_EVENTS" \
    "signal-processor" \
    "trading.signals" \
    "signal-processors"

# Create consumers for heartbeat events
echo "Setting up Heartbeat Event consumers..."
create_consumer_if_not_exists \
    "SYSTEM_EVENTS" \
    "heartbeat-processor" \
    "system.heartbeat" \
    "heartbeat-processors"

# Display stream and consumer information
echo ""
echo "=== NATS JetStream Configuration Summary ==="
echo ""
echo "Streams:"
nats --server="$NATS_SERVER" stream ls

echo ""
echo "Stream Details:"
for stream in "TRADING_EVENTS" "SYSTEM_EVENTS"; do
    echo ""
    echo "Stream: $stream"
    nats --server="$NATS_SERVER" stream info "$stream" --json | \
        jq -r '.config | "  Subjects: \(.subjects | join(", "))\n  Retention: \(.retention)\n  Max Age: \(.max_age)\n  Storage: \(.storage)"'
    
    echo "  Consumers:"
    nats --server="$NATS_SERVER" consumer ls "$stream" | sed 's/^/    /'
done

echo ""
echo "✅ NATS JetStream setup completed successfully!"
echo ""
echo "Available streams:"
echo "  - TRADING_EVENTS: Handles all trading-related events (orders, balance, signals, errors)"
echo "  - SYSTEM_EVENTS: Handles system events (heartbeats, monitoring)"
echo ""
echo "The streams are configured with:"
echo "  - Deduplication window: 2 minutes"
echo "  - Retention: Interest-based (messages kept until acknowledged)"
echo "  - Max delivery attempts: 3"
echo "  - Storage: File-based (persistent)"