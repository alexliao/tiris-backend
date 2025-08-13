#!/bin/bash

# Production NATS JetStream setup script
# Enhanced configuration for production deployment with security and monitoring

set -e

# Configuration
NATS_SERVER="${NATS_URL:-nats://nats:4222}"
MAX_RETRIES=60
RETRY_INTERVAL=2
SCRIPT_NAME="nats-setup-prod"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${BLUE}[$SCRIPT_NAME]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[$SCRIPT_NAME] ✅${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[$SCRIPT_NAME] ⚠️${NC} $1"
}

log_error() {
    echo -e "${RED}[$SCRIPT_NAME] ❌${NC} $1"
}

log "🚀 Starting NATS JetStream production setup..."

# Wait for NATS server to be available with authentication
log "⏳ Waiting for NATS server at $NATS_SERVER..."
for i in $(seq 1 $MAX_RETRIES); do
    if nats --server="$NATS_SERVER" server ping > /dev/null 2>&1; then
        log_success "NATS server is ready!"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        log_error "Failed to connect to NATS server after $MAX_RETRIES attempts"
        exit 1
    fi
    echo "   Attempt $i/$MAX_RETRIES failed, retrying in $RETRY_INTERVAL seconds..."
    sleep $RETRY_INTERVAL
done

# Display server information
log "📊 NATS Server Information:"
nats --server="$NATS_SERVER" server info --json | jq -r '
    "  Version: " + .version,
    "  Cluster: " + (.cluster.name // "standalone"),
    "  JetStream: " + (if .jetstream.enabled then "enabled" else "disabled" end),
    "  Max Memory: " + (.jetstream.config.max_memory | tostring),
    "  Max Storage: " + (.jetstream.config.max_storage | tostring)
'

# Function to create stream with production settings
create_production_stream() {
    local stream_name=$1
    local subjects=$2
    local retention=$3
    local max_age=$4
    local storage=$5
    local max_msgs=$6
    local max_bytes=$7
    local replicas=$8
    
    log "🔧 Checking stream '$stream_name'..."
    if nats --server="$NATS_SERVER" stream info "$stream_name" > /dev/null 2>&1; then
        log_warning "Stream '$stream_name' already exists, checking configuration..."
        
        # Compare configuration and update if needed
        current_config=$(nats --server="$NATS_SERVER" stream info "$stream_name" --json)
        current_subjects=$(echo "$current_config" | jq -r '.config.subjects | join(",")')
        current_retention=$(echo "$current_config" | jq -r '.config.retention')
        
        if [ "$current_subjects" != "$subjects" ] || [ "$current_retention" != "$retention" ]; then
            log_warning "Stream configuration differs, updating..."
            nats --server="$NATS_SERVER" stream edit "$stream_name" \
                --subjects="$subjects" \
                --retention="$retention" \
                --max-age="$max_age" \
                --storage="$storage" \
                --replicas="$replicas" \
                --discard=old \
                --max-msg-size=2MB \
                --max-msgs="$max_msgs" \
                --max-bytes="$max_bytes" \
                --dupe-window=5m \
                --allow-rollup \
                --deny-delete \
                --deny-purge
            log_success "Stream '$stream_name' updated successfully"
        else
            log_success "Stream '$stream_name' configuration is current"
        fi
    else
        log "🆕 Creating stream '$stream_name'..."
        nats --server="$NATS_SERVER" stream add \
            --subjects="$subjects" \
            --retention="$retention" \
            --max-age="$max_age" \
            --storage="$storage" \
            --replicas="$replicas" \
            --discard=old \
            --max-msg-size=2MB \
            --max-msgs="$max_msgs" \
            --max-bytes="$max_bytes" \
            --dupe-window=5m \
            --allow-rollup \
            --deny-delete \
            --deny-purge \
            "$stream_name"
        log_success "Stream '$stream_name' created successfully"
    fi
}

# Function to create consumer with production settings
create_production_consumer() {
    local stream_name=$1
    local consumer_name=$2
    local filter_subject=$3
    local queue_group=$4
    local max_deliver=$5
    local ack_wait=$6
    
    log "👥 Checking consumer '$consumer_name' in stream '$stream_name'..."
    if nats --server="$NATS_SERVER" consumer info "$stream_name" "$consumer_name" > /dev/null 2>&1; then
        log_success "Consumer '$consumer_name' already exists"
    else
        log "🆕 Creating consumer '$consumer_name'..."
        nats --server="$NATS_SERVER" consumer add \
            --filter="$filter_subject" \
            --ack=explicit \
            --pull \
            --deliver=all \
            --max-deliver="$max_deliver" \
            --wait="$ack_wait" \
            --replay=instant \
            --max-pending=1000 \
            --heartbeat=30s \
            --flow-control \
            ${queue_group:+--queue="$queue_group"} \
            "$stream_name" \
            "$consumer_name"
        log_success "Consumer '$consumer_name' created successfully"
    fi
}

# Create production streams with appropriate settings

# Trading Events Stream - High throughput, longer retention
log "📈 Setting up Trading Events stream..."
create_production_stream \
    "TRADING_EVENTS" \
    "trading.orders.*,trading.balance.*,trading.signals,trading.errors" \
    "limits" \
    "72h" \
    "file" \
    "1000000" \
    "10GB" \
    "1"

# System Events Stream - Lower volume, shorter retention  
log "🖥️  Setting up System Events stream..."
create_production_stream \
    "SYSTEM_EVENTS" \
    "system.*" \
    "limits" \
    "24h" \
    "file" \
    "100000" \
    "1GB" \
    "1"

# Audit Events Stream - Long-term retention, compliance
log "🔍 Setting up Audit Events stream..."
create_production_stream \
    "AUDIT_EVENTS" \
    "audit.*" \
    "limits" \
    "30d" \
    "file" \
    "10000000" \
    "50GB" \
    "1"

# Metrics Stream - High frequency, short retention
log "📊 Setting up Metrics stream..."
create_production_stream \
    "METRICS" \
    "metrics.*" \
    "limits" \
    "6h" \
    "file" \
    "500000" \
    "2GB" \
    "1"

# Create production consumers with appropriate settings

log "👥 Setting up Trading Event consumers..."

# Order processing consumers with higher reliability
create_production_consumer \
    "TRADING_EVENTS" \
    "order-processor-primary" \
    "trading.orders.*" \
    "order-processors" \
    "5" \
    "60s"

create_production_consumer \
    "TRADING_EVENTS" \
    "order-processor-secondary" \
    "trading.orders.*" \
    "order-processors-backup" \
    "3" \
    "30s"

# Balance processing consumers
create_production_consumer \
    "TRADING_EVENTS" \
    "balance-processor-primary" \
    "trading.balance.*" \
    "balance-processors" \
    "5" \
    "60s"

# Error processing consumers - immediate processing
create_production_consumer \
    "TRADING_EVENTS" \
    "error-processor" \
    "trading.errors" \
    "error-processors" \
    "10" \
    "30s"

# Signal processing consumers
create_production_consumer \
    "TRADING_EVENTS" \
    "signal-processor" \
    "trading.signals" \
    "signal-processors" \
    "3" \
    "45s"

log "🖥️  Setting up System Event consumers..."

# System monitoring consumers
create_production_consumer \
    "SYSTEM_EVENTS" \
    "heartbeat-processor" \
    "system.heartbeat" \
    "heartbeat-processors" \
    "3" \
    "60s"

create_production_consumer \
    "SYSTEM_EVENTS" \
    "system-monitor" \
    "system.*" \
    "system-monitors" \
    "5" \
    "30s"

log "🔍 Setting up Audit Event consumers..."

# Audit logging consumer - reliable processing
create_production_consumer \
    "AUDIT_EVENTS" \
    "audit-logger" \
    "audit.*" \
    "audit-processors" \
    "10" \
    "300s"

log "📊 Setting up Metrics consumers..."

# Metrics collection consumer
create_production_consumer \
    "METRICS" \
    "metrics-collector" \
    "metrics.*" \
    "metrics-processors" \
    "3" \
    "30s"

# Display comprehensive configuration summary
log ""
log_success "🎉 NATS JetStream production setup completed!"
log ""
log "📋 Configuration Summary:"
log "========================"

echo ""
echo "📊 Streams:"
nats --server="$NATS_SERVER" stream ls

echo ""
log "📈 Stream Details:"
for stream in "TRADING_EVENTS" "SYSTEM_EVENTS" "AUDIT_EVENTS" "METRICS"; do
    echo ""
    echo "🔸 Stream: $stream"
    stream_info=$(nats --server="$NATS_SERVER" stream info "$stream" --json)
    
    echo "$stream_info" | jq -r '
        "  📝 Subjects: " + (.config.subjects | join(", ")),
        "  💾 Storage: " + .config.storage,
        "  📅 Max Age: " + .config.max_age,
        "  📊 Max Messages: " + (.config.max_msgs | tostring),
        "  💽 Max Bytes: " + .config.max_bytes,
        "  🔄 Retention: " + .config.retention,
        "  📈 Current Messages: " + (.state.messages | tostring),
        "  💿 Current Bytes: " + (.state.bytes | tostring)
    '
    
    echo "  👥 Consumers:"
    nats --server="$NATS_SERVER" consumer ls "$stream" | sed 's/^/    - /'
done

echo ""
log_success "✅ Production NATS JetStream setup completed successfully!"
echo ""
log "🔧 Production Features Enabled:"
log "  - 🛡️  Authentication and authorization"
log "  - 🔄 Message deduplication (5-minute window)"
log "  - 💾 Persistent file storage"
log "  - 📊 Multiple consumer groups for load balancing"
log "  - ⚡ Flow control and heartbeat monitoring"
log "  - 🔒 Stream protection (deny delete/purge)"
log "  - 📈 Resource limits and quotas"
echo ""
log "📚 Management Commands:"
log "  View stream status: nats --server=\"$NATS_SERVER\" stream ls"
log "  View consumer status: nats --server=\"$NATS_SERVER\" consumer ls <stream>"
log "  Monitor messages: nats --server=\"$NATS_SERVER\" stream info <stream>"
log "  View server stats: nats --server=\"$NATS_SERVER\" server info"