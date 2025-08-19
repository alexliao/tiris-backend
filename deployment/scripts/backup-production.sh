#!/bin/bash
set -e

# Production Backup Script for Tiris Backend
# Handles database backups, log archival, and system maintenance

echo "🔄 Starting Tiris Backend production backup..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BACKUP_DIR="/opt/tiris/backups"
LOG_DIR="/opt/tiris/logs"
RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-30}
COMPRESS_LOGS_DAYS=${COMPRESS_LOGS_DAYS:-7}
S3_BUCKET=${BACKUP_S3_BUCKET:-}
NOTIFICATION_WEBHOOK=${BACKUP_NOTIFICATION_WEBHOOK:-}

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" >> "$BACKUP_DIR/backup.log"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1" >> "$BACKUP_DIR/backup.log"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1" >> "$BACKUP_DIR/backup.log"
    send_notification "❌ Backup failed: $1"
    exit 1
}

# Function to send notifications
send_notification() {
    local message="$1"
    if [[ -n "$NOTIFICATION_WEBHOOK" ]]; then
        curl -X POST "$NOTIFICATION_WEBHOOK" \
            -H 'Content-Type: application/json' \
            -d "{\"text\":\"$message\"}" \
            --silent --max-time 10 || true
    fi
}

# Function to get database connection info from environment
get_db_config() {
    if [[ -f "/opt/tiris/tiris-backend/.env.prod" ]]; then
        source /opt/tiris/tiris-backend/.env.prod
        DB_HOST=${DB_HOST:-postgres}
        DB_PORT=${DB_PORT:-5432}
        DB_NAME=${DB_NAME:-tiris_prod}
        DB_USER=${DB_USER:-tiris_user}
        export PGPASSWORD="$DB_PASSWORD"
    else
        error "Environment file not found: /opt/tiris/tiris-backend/.env.prod"
    fi
}

# Function to create backup directories
setup_backup_dirs() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local date_dir=$(date +%Y-%m-%d)
    
    BACKUP_DATE_DIR="$BACKUP_DIR/$date_dir"
    BACKUP_TIMESTAMP_DIR="$BACKUP_DATE_DIR/$timestamp"
    
    mkdir -p "$BACKUP_TIMESTAMP_DIR"/{database,logs,config,monitoring}
    
    log "Backup directory created: $BACKUP_TIMESTAMP_DIR"
}

# Function to backup database
backup_database() {
    log "Starting database backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_TIMESTAMP_DIR/database/tiris_db_$timestamp.sql"
    local backup_file_gz="$backup_file.gz"
    
    # Create database dump
    if docker exec tiris-postgres-prod pg_dump \
        -h localhost \
        -p 5432 \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --verbose \
        --clean \
        --create \
        --if-exists > "$backup_file"; then
        
        # Compress the backup
        gzip "$backup_file"
        
        # Verify backup integrity
        if gzip -t "$backup_file_gz"; then
            local size=$(du -h "$backup_file_gz" | cut -f1)
            log "Database backup completed successfully: $backup_file_gz ($size)"
            
            # Create checksum
            md5sum "$backup_file_gz" > "$backup_file_gz.md5"
            
            # Upload to S3 if configured
            if [[ -n "$S3_BUCKET" ]] && command -v aws &> /dev/null; then
                log "Uploading backup to S3..."
                if aws s3 cp "$backup_file_gz" "s3://$S3_BUCKET/database/$(basename "$backup_file_gz")"; then
                    log "Backup uploaded to S3 successfully"
                else
                    warn "Failed to upload backup to S3"
                fi
            fi
            
        else
            error "Backup file is corrupted: $backup_file_gz"
        fi
    else
        error "Database backup failed"
    fi
}

# Function to backup application logs
backup_logs() {
    log "Starting log backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_archive="$BACKUP_TIMESTAMP_DIR/logs/tiris_logs_$timestamp.tar.gz"
    
    # Create logs archive
    if tar -czf "$log_archive" -C "$LOG_DIR" . --exclude="*.gz" --exclude="backup.log"; then
        local size=$(du -h "$log_archive" | cut -f1)
        log "Log backup completed: $log_archive ($size)"
        
        # Compress old logs
        find "$LOG_DIR" -name "*.log" -mtime +$COMPRESS_LOGS_DAYS -exec gzip {} \;
        
    else
        warn "Log backup failed"
    fi
}

# Function to backup configuration files
backup_config() {
    log "Starting configuration backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local config_archive="$BACKUP_TIMESTAMP_DIR/config/tiris_config_$timestamp.tar.gz"
    
    # List of important configuration files and directories
    local config_paths=(
        "/opt/tiris/tiris-backend/.env.prod"
        "/opt/tiris/tiris-backend/docker-compose.prod.yml"
        "/opt/tiris/tiris-backend/config"
        "/etc/nginx/sites-available"
        "/etc/nginx/nginx.conf"
        "/etc/letsencrypt"
        "/etc/crontab"
    )
    
    # Create configuration archive
    local existing_paths=()
    for path in "${config_paths[@]}"; do
        if [[ -e "$path" ]]; then
            existing_paths+=("$path")
        fi
    done
    
    if [[ ${#existing_paths[@]} -gt 0 ]]; then
        if tar -czf "$config_archive" "${existing_paths[@]}" 2>/dev/null; then
            local size=$(du -h "$config_archive" | cut -f1)
            log "Configuration backup completed: $config_archive ($size)"
        else
            warn "Configuration backup failed"
        fi
    else
        warn "No configuration files found to backup"
    fi
}

# Function to backup monitoring data
backup_monitoring() {
    log "Starting monitoring data backup..."
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local monitoring_archive="$BACKUP_TIMESTAMP_DIR/monitoring/tiris_monitoring_$timestamp.tar.gz"
    
    # Prometheus data directory (if running)
    local prometheus_data="/opt/tiris/data/prometheus"
    
    if [[ -d "$prometheus_data" ]]; then
        if tar -czf "$monitoring_archive" -C "$prometheus_data" . 2>/dev/null; then
            local size=$(du -h "$monitoring_archive" | cut -f1)
            log "Monitoring backup completed: $monitoring_archive ($size)"
        else
            warn "Monitoring backup failed"
        fi
    else
        log "No monitoring data to backup"
    fi
}

# Function to cleanup old backups
cleanup_old_backups() {
    log "Cleaning up backups older than $RETENTION_DAYS days..."
    
    # Remove old backup directories
    find "$BACKUP_DIR" -type d -name "20*" -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true
    
    # Remove old individual backup files
    find "$BACKUP_DIR" -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "*.tar.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    find "$BACKUP_DIR" -name "*.md5" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    
    # Clean up log files
    find "$BACKUP_DIR" -name "backup.log" -size +10M -exec truncate -s 0 {} \; 2>/dev/null || true
    
    log "Cleanup completed"
}

# Function to perform system maintenance
system_maintenance() {
    log "Starting system maintenance..."
    
    # Docker system cleanup
    if command -v docker &> /dev/null; then
        log "Cleaning up Docker system..."
        docker system prune -f --volumes || warn "Docker cleanup failed"
    fi
    
    # Update package lists (if running as root)
    if [[ $EUID -eq 0 ]]; then
        log "Updating package lists..."
        apt update &>/dev/null || warn "Package list update failed"
    fi
    
    # Rotate logs
    if command -v logrotate &> /dev/null; then
        log "Rotating logs..."
        logrotate -f /etc/logrotate.conf &>/dev/null || warn "Log rotation failed"
    fi
    
    log "System maintenance completed"
}

# Function to generate backup report
generate_backup_report() {
    local report_file="$BACKUP_TIMESTAMP_DIR/backup_report.txt"
    
    cat > "$report_file" << EOF
# Tiris Backend Backup Report
Generated: $(date)
Backup Directory: $BACKUP_TIMESTAMP_DIR
Retention Period: $RETENTION_DAYS days

## Backup Contents:
$(find "$BACKUP_TIMESTAMP_DIR" -type f -exec ls -lh {} \; | awk '{print $9 " (" $5 ")"}')

## System Information:
Disk Usage: $(df -h /opt/tiris | tail -1)
Available Space: $(df -h /opt/tiris | tail -1 | awk '{print $4}')
Memory Usage: $(free -h | grep '^Mem:' | awk '{print $3 "/" $2}')

## Container Status:
$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Size}}" | grep tiris || echo "No containers running")

## Database Size:
$(docker exec tiris-postgres-prod psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT pg_size_pretty(pg_database_size('$DB_NAME'));" -t 2>/dev/null | xargs || echo "Unable to determine database size")

EOF

    log "Backup report generated: $report_file"
}

# Function to verify backup integrity
verify_backups() {
    log "Verifying backup integrity..."
    
    # Verify database backup
    local db_backup=$(find "$BACKUP_TIMESTAMP_DIR/database" -name "*.sql.gz" | head -1)
    if [[ -n "$db_backup" && -f "$db_backup.md5" ]]; then
        if md5sum -c "$db_backup.md5" &>/dev/null; then
            log "Database backup integrity verified"
        else
            error "Database backup integrity check failed"
        fi
    fi
    
    # Verify other archives
    for archive in "$BACKUP_TIMESTAMP_DIR"/*/*.tar.gz; do
        if [[ -f "$archive" ]]; then
            if tar -tzf "$archive" &>/dev/null; then
                log "Archive integrity verified: $(basename "$archive")"
            else
                warn "Archive integrity check failed: $(basename "$archive")"
            fi
        fi
    done
}

# Main backup execution
main() {
    local start_time=$(date +%s)
    
    log "🚀 Starting Tiris Backend backup process..."
    
    # Setup
    get_db_config
    setup_backup_dirs
    
    # Perform backups
    backup_database
    backup_logs
    backup_config
    backup_monitoring
    
    # Verify backups
    verify_backups
    
    # Generate report
    generate_backup_report
    
    # Maintenance
    system_maintenance
    cleanup_old_backups
    
    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log "✅ Backup process completed successfully in ${duration}s"
    
    # Send success notification
    send_notification "✅ Tiris Backend backup completed successfully in ${duration}s"
    
    # Display summary
    echo ""
    echo "=== Backup Summary ==="
    echo "📁 Backup Location: $BACKUP_TIMESTAMP_DIR"
    echo "📊 Total Size: $(du -sh "$BACKUP_TIMESTAMP_DIR" | cut -f1)"
    echo "⏱️  Duration: ${duration}s"
    echo "🗓️  Retention: $RETENTION_DAYS days"
    echo ""
    echo "=== Quick Commands ==="
    echo "📋 View report: cat $BACKUP_TIMESTAMP_DIR/backup_report.txt"
    echo "🔍 List backups: ls -la $BACKUP_TIMESTAMP_DIR"
    echo "📈 Check space: df -h /opt/tiris"
}

# Error handling
trap 'error "Backup script interrupted"' INT TERM

# Check if running with proper permissions
if [[ ! -w "$BACKUP_DIR" ]]; then
    error "No write permission to backup directory: $BACKUP_DIR"
fi

# Execute main function
main "$@"