#!/bin/bash

# Production PostgreSQL backup script with rotation and compression
# Supports both full database dumps and selective table backups

set -e

# Configuration
BACKUP_DIR="/backups"
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-tiris_prod}"
DB_USER="${DB_USER:-tiris_user}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
COMPRESS_BACKUPS="${COMPRESS_BACKUPS:-true}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$BACKUP_DIR/backup_${TIMESTAMP}.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() {
    echo -e "${BLUE}[BACKUP]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[BACKUP] ✅${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[BACKUP] ⚠️${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[BACKUP] ❌${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

log "🚀 Starting PostgreSQL backup process..."
log "📊 Configuration:"
log "  Database: $DB_NAME@$DB_HOST:$DB_PORT"
log "  User: $DB_USER"
log "  Backup Directory: $BACKUP_DIR"
log "  Retention: $RETENTION_DAYS days"
log "  Compression: $COMPRESS_BACKUPS"

# Check database connectivity
log "🔍 Checking database connectivity..."
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
    log_error "Cannot connect to database $DB_NAME@$DB_HOST:$DB_PORT"
    exit 1
fi
log_success "Database connection successful"

# Function to perform full database backup
backup_full_database() {
    local backup_file="$BACKUP_DIR/full_backup_${DB_NAME}_${TIMESTAMP}.sql"
    
    log "📦 Creating full database backup..."
    
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --verbose \
        --format=custom \
        --compress=9 \
        --create \
        --clean \
        --if-exists \
        --no-owner \
        --no-privileges \
        "$DB_NAME" > "${backup_file}.dump" 2>> "$LOG_FILE"; then
        
        log_success "Full backup created: ${backup_file}.dump"
        
        # Also create plain SQL backup for easier inspection
        if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
            --verbose \
            --format=plain \
            --create \
            --clean \
            --if-exists \
            --no-owner \
            --no-privileges \
            "$DB_NAME" > "$backup_file" 2>> "$LOG_FILE"; then
            
            log_success "SQL backup created: $backup_file"
            
            # Compress SQL backup if enabled
            if [ "$COMPRESS_BACKUPS" = "true" ]; then
                gzip "$backup_file"
                log_success "SQL backup compressed: ${backup_file}.gz"
            fi
        else
            log_error "Failed to create SQL backup"
            return 1
        fi
        
        # Get backup size information
        local dump_size=$(ls -lh "${backup_file}.dump" | awk '{print $5}')
        local sql_size
        if [ "$COMPRESS_BACKUPS" = "true" ]; then
            sql_size=$(ls -lh "${backup_file}.gz" | awk '{print $5}')
        else
            sql_size=$(ls -lh "$backup_file" | awk '{print $5}')
        fi
        
        log "📊 Backup sizes:"
        log "  Custom format: $dump_size"
        log "  SQL format: $sql_size"
        
        return 0
    else
        log_error "Failed to create full database backup"
        return 1
    fi
}

# Function to perform schema-only backup
backup_schema_only() {
    local backup_file="$BACKUP_DIR/schema_backup_${DB_NAME}_${TIMESTAMP}.sql"
    
    log "🏗️  Creating schema-only backup..."
    
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --verbose \
        --schema-only \
        --format=plain \
        --create \
        --clean \
        --if-exists \
        --no-owner \
        --no-privileges \
        "$DB_NAME" > "$backup_file" 2>> "$LOG_FILE"; then
        
        log_success "Schema backup created: $backup_file"
        
        if [ "$COMPRESS_BACKUPS" = "true" ]; then
            gzip "$backup_file"
            log_success "Schema backup compressed: ${backup_file}.gz"
        fi
        
        return 0
    else
        log_error "Failed to create schema backup"
        return 1
    fi
}

# Function to backup specific tables (critical data)
backup_critical_tables() {
    local backup_file="$BACKUP_DIR/critical_tables_${DB_NAME}_${TIMESTAMP}.sql"
    local critical_tables=("users" "tradings" "sub_accounts" "transactions" "trading_logs")
    
    log "🔐 Creating critical tables backup..."
    
    # Build table list for pg_dump
    local table_args=""
    for table in "${critical_tables[@]}"; do
        table_args="$table_args --table=$table"
    done
    
    if pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --verbose \
        --format=custom \
        --compress=9 \
        --data-only \
        --no-owner \
        --no-privileges \
        $table_args \
        "$DB_NAME" > "${backup_file}.dump" 2>> "$LOG_FILE"; then
        
        log_success "Critical tables backup created: ${backup_file}.dump"
        
        # Get list of backed up tables
        log "📋 Backed up tables:"
        for table in "${critical_tables[@]}"; do
            local count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | tr -d ' ')
            log "  - $table: $count rows"
        done
        
        return 0
    else
        log_error "Failed to create critical tables backup"
        return 1
    fi
}

# Function to create a backup manifest
create_backup_manifest() {
    local manifest_file="$BACKUP_DIR/backup_manifest_${TIMESTAMP}.json"
    
    log "📋 Creating backup manifest..."
    
    cat > "$manifest_file" << EOF
{
    "timestamp": "$TIMESTAMP",
    "database": {
        "name": "$DB_NAME",
        "host": "$DB_HOST",
        "port": "$DB_PORT",
        "user": "$DB_USER"
    },
    "backup_info": {
        "retention_days": $RETENTION_DAYS,
        "compression_enabled": $COMPRESS_BACKUPS,
        "backup_directory": "$BACKUP_DIR"
    },
    "files": [
EOF

    # List all backup files created in this session
    local files=($(ls -1 "$BACKUP_DIR"/*_${TIMESTAMP}.* 2>/dev/null || true))
    local file_count=${#files[@]}
    local file_index=0
    
    for file in "${files[@]}"; do
        file_index=$((file_index + 1))
        local size=$(ls -lh "$file" | awk '{print $5}')
        local filename=$(basename "$file")
        
        cat >> "$manifest_file" << EOF
        {
            "filename": "$filename",
            "size": "$size",
            "path": "$file",
            "type": "backup"
        }$([ $file_index -lt $file_count ] && echo "," || echo "")
EOF
    done

    cat >> "$manifest_file" << EOF
    ],
    "database_stats": {
$(
    # Get database statistics
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            '        \"total_size\": \"' || pg_size_pretty(pg_database_size('$DB_NAME')) || '\",'
    " 2>/dev/null || echo '        "total_size": "unknown",'
)
$(
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
        SELECT 
            '        \"table_count\": ' || count(*) || ','
        FROM information_schema.tables 
        WHERE table_schema = 'public'
    " 2>/dev/null || echo '        "table_count": 0,'
)
        "backup_completion": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    }
}
EOF

    log_success "Backup manifest created: $manifest_file"
}

# Function to clean up old backups
cleanup_old_backups() {
    log "🧹 Cleaning up backups older than $RETENTION_DAYS days..."
    
    local deleted_count=0
    local total_size_freed=0
    
    # Find and delete old backup files
    while IFS= read -r -d '' file; do
        local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo 0)
        total_size_freed=$((total_size_freed + size))
        rm "$file"
        deleted_count=$((deleted_count + 1))
        log "  Deleted: $(basename "$file")"
    done < <(find "$BACKUP_DIR" -name "*backup*.sql*" -o -name "*backup*.dump" -o -name "*manifest*.json" -o -name "*backup*.log" -type f -mtime +$RETENTION_DAYS -print0 2>/dev/null)
    
    if [ $deleted_count -gt 0 ]; then
        local size_freed_mb=$((total_size_freed / 1024 / 1024))
        log_success "Cleaned up $deleted_count old files, freed ${size_freed_mb}MB"
    else
        log "No old backups found to clean up"
    fi
}

# Function to verify backup integrity
verify_backup() {
    local backup_file="$1"
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    log "🔍 Verifying backup integrity: $(basename "$backup_file")"
    
    # For custom format backups, use pg_restore to validate
    if [[ "$backup_file" == *.dump ]]; then
        if pg_restore --list "$backup_file" > /dev/null 2>&1; then
            log_success "Backup file is valid and readable"
            return 0
        else
            log_error "Backup file appears to be corrupted"
            return 1
        fi
    fi
    
    # For SQL files, check if they're valid SQL
    if [[ "$backup_file" == *.sql || "$backup_file" == *.sql.gz ]]; then
        local check_file="$backup_file"
        if [[ "$backup_file" == *.gz ]]; then
            check_file="/tmp/check_backup.sql"
            gunzip -c "$backup_file" > "$check_file"
        fi
        
        if grep -q "PostgreSQL database dump" "$check_file" 2>/dev/null; then
            log_success "SQL backup file appears valid"
            [ "$check_file" != "$backup_file" ] && rm -f "$check_file"
            return 0
        else
            log_error "SQL backup file may be invalid"
            [ "$check_file" != "$backup_file" ] && rm -f "$check_file"
            return 1
        fi
    fi
    
    log_warning "Cannot verify backup format: $(basename "$backup_file")"
    return 0
}

# Main backup execution
main() {
    local success=true
    
    # Perform different types of backups
    if ! backup_full_database; then
        success=false
    fi
    
    if ! backup_schema_only; then
        success=false
    fi
    
    if ! backup_critical_tables; then
        success=false
    fi
    
    # Create backup manifest
    create_backup_manifest
    
    # Verify backups
    log "🔍 Verifying created backups..."
    for backup_file in "$BACKUP_DIR"/*_${TIMESTAMP}.*; do
        if [ -f "$backup_file" ] && [[ "$backup_file" != *".log" ]] && [[ "$backup_file" != *".json" ]]; then
            verify_backup "$backup_file"
        fi
    done
    
    # Clean up old backups
    cleanup_old_backups
    
    # Final summary
    log ""
    if $success; then
        log_success "🎉 Backup process completed successfully!"
    else
        log_error "⚠️ Backup process completed with errors!"
    fi
    
    log "📊 Final Summary:"
    log "  Timestamp: $TIMESTAMP"
    log "  Log file: $LOG_FILE"
    log "  Total backup files: $(ls -1 "$BACKUP_DIR"/*_${TIMESTAMP}.* 2>/dev/null | grep -v ".log" | wc -l | tr -d ' ')"
    log "  Backup directory: $BACKUP_DIR"
    
    # Show disk usage
    local backup_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1 || echo "unknown")
    log "  Total backup directory size: $backup_size"
    
    return $([ $success = true ] && echo 0 || echo 1)
}

# Execute main function
main "$@"