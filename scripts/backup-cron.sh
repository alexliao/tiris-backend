#!/bin/bash

# Cron-based backup service for production PostgreSQL
# Runs automated backups on schedule with proper logging and error handling

set -e

# Configuration from environment variables
BACKUP_SCHEDULE="${BACKUP_SCHEDULE:-0 2 * * *}"  # Default: Daily at 2 AM
BACKUP_SCRIPT="/scripts/backup-db.sh"
CRON_LOG="/var/log/cron.log"
BACKUP_LOG_DIR="/backups/logs"

# Colors for output (disabled in cron environment)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Logging functions
log() {
    echo -e "${BLUE}[BACKUP-CRON]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[BACKUP-CRON] ✅${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[BACKUP-CRON] ⚠️${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[BACKUP-CRON] ❌${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Create log directory
mkdir -p "$BACKUP_LOG_DIR"

# Function to setup cron job
setup_cron() {
    log "🕒 Setting up backup cron job..."
    log "Schedule: $BACKUP_SCHEDULE"
    
    # Create cron job entry
    cat > /tmp/backup-crontab << EOF
# Automated PostgreSQL backup for Tiris Backend
# Generated on $(date)
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
MAILTO=""

# Environment variables for backup
DB_HOST=${DB_HOST:-postgres}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-tiris_prod}
DB_USER=${DB_USER:-tiris_user}
PGPASSWORD=${PGPASSWORD:-}
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-7}
COMPRESS_BACKUPS=${COMPRESS_BACKUPS:-true}

# Backup schedule: $BACKUP_SCHEDULE
$BACKUP_SCHEDULE /scripts/backup-db.sh >> $BACKUP_LOG_DIR/cron.log 2>&1

# Health check - run every hour to ensure backup service is alive
0 * * * * echo "\$(date): Backup service health check" >> $BACKUP_LOG_DIR/health.log

# Weekly backup log rotation (Sunday at 1 AM)
0 1 * * 0 find $BACKUP_LOG_DIR -name "*.log" -mtime +30 -delete 2>/dev/null || true

EOF

    # Install cron job
    crontab /tmp/backup-crontab
    rm -f /tmp/backup-crontab
    
    log_success "Cron job installed successfully"
    
    # Show current cron jobs
    log "📋 Current cron schedule:"
    crontab -l | grep -v "^#" | while read line; do
        [ -n "$line" ] && log "  $line"
    done
}

# Function to run immediate backup (for testing)
run_immediate_backup() {
    log "🚀 Running immediate backup for testing..."
    
    if [ -x "$BACKUP_SCRIPT" ]; then
        log "Executing: $BACKUP_SCRIPT"
        if $BACKUP_SCRIPT; then
            log_success "Immediate backup completed successfully"
        else
            log_error "Immediate backup failed"
            return 1
        fi
    else
        log_error "Backup script not found or not executable: $BACKUP_SCRIPT"
        return 1
    fi
}

# Function to check backup health
check_backup_health() {
    log "🏥 Checking backup service health..."
    
    # Check if cron is running
    if pgrep cron >/dev/null 2>&1 || pgrep crond >/dev/null 2>&1; then
        log_success "Cron service is running"
    else
        log_error "Cron service is not running"
        return 1
    fi
    
    # Check if backup script exists and is executable
    if [ -x "$BACKUP_SCRIPT" ]; then
        log_success "Backup script is available and executable"
    else
        log_error "Backup script not found or not executable: $BACKUP_SCRIPT"
        return 1
    fi
    
    # Check recent backup activity
    local recent_backups=$(find /backups -name "*backup*.dump" -mtime -2 2>/dev/null | wc -l | tr -d ' ')
    if [ "$recent_backups" -gt 0 ]; then
        log_success "Found $recent_backups recent backup files (last 2 days)"
    else
        log_warning "No recent backup files found (last 2 days)"
    fi
    
    # Check log files
    if [ -f "$BACKUP_LOG_DIR/cron.log" ]; then
        local log_size=$(ls -lh "$BACKUP_LOG_DIR/cron.log" | awk '{print $5}')
        log "📋 Cron log file: $log_size"
        
        # Show last few log entries
        if [ -s "$BACKUP_LOG_DIR/cron.log" ]; then
            log "📝 Recent log entries:"
            tail -5 "$BACKUP_LOG_DIR/cron.log" | sed 's/^/  /'
        fi
    else
        log_warning "No cron log file found yet"
    fi
    
    # Check disk space
    local backup_dir_usage=$(df -h /backups 2>/dev/null | tail -1 | awk '{print $5}' | sed 's/%//')
    if [ -n "$backup_dir_usage" ] && [ "$backup_dir_usage" -lt 90 ]; then
        log_success "Backup directory disk usage: ${backup_dir_usage}%"
    else
        log_warning "Backup directory disk usage: ${backup_dir_usage}% (high)"
    fi
    
    log_success "Health check completed"
}

# Function to show backup status
show_backup_status() {
    log "📊 Backup Service Status Report"
    log "================================="
    
    # Current time and schedule
    log "⏰ Current time: $(date)"
    log "📅 Backup schedule: $BACKUP_SCHEDULE"
    
    # Next scheduled run
    if command -v cron-expression-next >/dev/null 2>&1; then
        local next_run=$(cron-expression-next "$BACKUP_SCHEDULE" 2>/dev/null || echo "Unknown")
        log "⏭️  Next scheduled backup: $next_run"
    fi
    
    # Recent backups
    log "📦 Recent backup files:"
    find /backups -name "*backup*.dump" -mtime -7 2>/dev/null | head -10 | while read file; do
        local size=$(ls -lh "$file" | awk '{print $5}')
        local date=$(ls -l "$file" | awk '{print $6, $7, $8}')
        log "  $(basename "$file") - $size - $date"
    done
    
    # Storage information
    local total_backups=$(find /backups -name "*backup*" -type f 2>/dev/null | wc -l | tr -d ' ')
    local backup_dir_size=$(du -sh /backups 2>/dev/null | cut -f1 || echo "unknown")
    log "💾 Storage: $total_backups files, $backup_dir_size total"
    
    # Service status
    check_backup_health
}

# Function to start cron service
start_cron_service() {
    log "🚀 Starting backup cron service..."
    
    # Setup cron job
    setup_cron
    
    # Start cron daemon based on the system
    if command -v cron >/dev/null 2>&1; then
        log "Starting cron daemon..."
        cron -f &
        CRON_PID=$!
        log_success "Cron daemon started with PID: $CRON_PID"
    elif command -v crond >/dev/null 2>&1; then
        log "Starting crond daemon..."
        crond -f &
        CRON_PID=$!
        log_success "Crond daemon started with PID: $CRON_PID"
    else
        log_error "No cron daemon found (cron or crond)"
        exit 1
    fi
    
    # Setup signal handlers
    trap "log 'Received SIGTERM, stopping cron...'; kill $CRON_PID; exit 0" TERM
    trap "log 'Received SIGINT, stopping cron...'; kill $CRON_PID; exit 0" INT
    
    # Run health check every 5 minutes
    while true; do
        sleep 300  # 5 minutes
        if ! kill -0 $CRON_PID 2>/dev/null; then
            log_error "Cron daemon has stopped unexpectedly"
            exit 1
        fi
        echo "$(date): Backup cron service is running (PID: $CRON_PID)" >> "$BACKUP_LOG_DIR/health.log"
    done
}

# Main function
main() {
    local command=${1:-start}
    
    case "$command" in
        "start")
            log "🏁 Starting backup cron service..."
            start_cron_service
            ;;
        "test"|"run")
            log "🧪 Running test backup..."
            run_immediate_backup
            ;;
        "health"|"check")
            check_backup_health
            ;;
        "status")
            show_backup_status
            ;;
        "setup")
            setup_cron
            ;;
        "help"|"--help")
            echo "Backup Cron Service for Tiris Backend"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  start    - Start the cron service (default)"
            echo "  test     - Run immediate test backup"
            echo "  health   - Check service health"
            echo "  status   - Show backup status report"
            echo "  setup    - Setup cron job only"
            echo "  help     - Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  BACKUP_SCHEDULE         - Cron schedule (default: '0 2 * * *')"
            echo "  BACKUP_RETENTION_DAYS   - Backup retention in days (default: 7)"
            echo "  COMPRESS_BACKUPS        - Enable compression (default: true)"
            echo "  DB_HOST, DB_PORT, DB_NAME, DB_USER - Database connection"
            ;;
        *)
            log_error "Unknown command: $command"
            log "Use '$0 help' to see available commands"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"