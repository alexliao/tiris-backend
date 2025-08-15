#!/bin/bash

# Tiris Backend - Port Availability Checker
# This script checks if all required ports are available for the project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to get service name for port
get_service_name() {
    case $1 in
        8080) echo "API Server" ;;
        2345) echo "Delve Debugger" ;;
        5432) echo "PostgreSQL/TimescaleDB" ;;
        6379) echo "Redis Cache" ;;
        4222) echo "NATS Client Protocol" ;;
        8222) echo "NATS HTTP Monitoring" ;;
        3000) echo "Grafana Dashboard" ;;
        9090) echo "Prometheus Server" ;;
        9093) echo "AlertManager" ;;
        3100) echo "Loki Log Aggregation" ;;
        9080) echo "Promtail Log Collection" ;;
        80) echo "HTTP (Production)" ;;
        443) echo "HTTPS/TLS (Production)" ;;
        *) echo "Unknown Service" ;;
    esac
}

# Required ports for development
DEV_PORTS=(8080 5432 6379 4222 8222)

# Required ports for monitoring
MONITORING_PORTS=(3000 9090 9093 3100 9080)

# Required ports for production
PROD_PORTS=(80 443 5432 6379 4222)

# Function to check if a port is in use
check_port() {
    local port=$1
    local service_name=$(get_service_name $port)
    
    if command -v lsof >/dev/null 2>&1; then
        # Use lsof if available (macOS/Linux)
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            local pid=$(lsof -Pi :$port -sTCP:LISTEN -t)
            local process=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
            echo -e "${RED}✗${NC} Port $port ($service_name) - ${RED}IN USE${NC} by PID $pid ($process)"
            return 1
        else
            echo -e "${GREEN}✓${NC} Port $port ($service_name) - ${GREEN}AVAILABLE${NC}"
            return 0
        fi
    elif command -v netstat >/dev/null 2>&1; then
        # Fallback to netstat (Linux)
        if netstat -tuln | grep -q ":$port "; then
            echo -e "${RED}✗${NC} Port $port ($service_name) - ${RED}IN USE${NC}"
            return 1
        else
            echo -e "${GREEN}✓${NC} Port $port ($service_name) - ${GREEN}AVAILABLE${NC}"
            return 0
        fi
    elif command -v ss >/dev/null 2>&1; then
        # Modern Linux alternative
        if ss -tuln | grep -q ":$port "; then
            echo -e "${RED}✗${NC} Port $port ($service_name) - ${RED}IN USE${NC}"
            return 1
        else
            echo -e "${GREEN}✓${NC} Port $port ($service_name) - ${GREEN}AVAILABLE${NC}"
            return 0
        fi
    else
        echo -e "${YELLOW}?${NC} Port $port ($service_name) - ${YELLOW}CANNOT CHECK${NC} (no lsof/netstat/ss available)"
        return 2
    fi
}

# Function to check a group of ports
check_port_group() {
    local group_name=$1
    shift
    local ports=("$@")
    local available=0
    local total=${#ports[@]}
    
    echo -e "\n${BLUE}=== $group_name ===${NC}"
    
    for port in "${ports[@]}"; do
        if check_port $port; then
            ((available++))
        fi
    done
    
    echo -e "\n${BLUE}Summary:${NC} $available/$total ports available"
    
    if [ $available -eq $total ]; then
        echo -e "${GREEN}✓ All $group_name ports are available${NC}"
        return 0
    else
        echo -e "${RED}✗ Some $group_name ports are in use${NC}"
        return 1
    fi
}

# Function to kill processes on specific ports
kill_port_processes() {
    local ports=("$@")
    
    echo -e "\n${YELLOW}Attempting to free up ports...${NC}"
    
    for port in "${ports[@]}"; do
        if command -v lsof >/dev/null 2>&1; then
            local pids=$(lsof -Pi :$port -sTCP:LISTEN -t 2>/dev/null || true)
            if [ -n "$pids" ]; then
                echo -e "Killing processes on port $port: $pids"
                echo $pids | xargs kill -9 2>/dev/null || true
                sleep 1
            fi
        fi
    done
    
    echo -e "${GREEN}Port cleanup completed${NC}"
}

# Function to show detailed port usage
show_detailed_usage() {
    echo -e "\n${BLUE}=== Detailed Port Usage ===${NC}"
    
    if command -v lsof >/dev/null 2>&1; then
        echo -e "\n${YELLOW}Active listeners on project ports:${NC}"
        local all_ports=(8080 2345 5432 6379 4222 8222 3000 9090 9093 3100 9080 80 443)
        for port in "${all_ports[@]}"; do
            local result=$(lsof -Pi :$port -sTCP:LISTEN 2>/dev/null || true)
            if [ -n "$result" ]; then
                echo -e "\n${RED}Port $port ($(get_service_name $port)):${NC}"
                echo "$result" | awk 'NR>1 {printf "  PID %-8s CMD: %s\n", $2, $1}'
            fi
        done
    fi
    
    # Show Docker containers using these ports
    if command -v docker >/dev/null 2>&1; then
        echo -e "\n${YELLOW}Docker containers with exposed ports:${NC}"
        docker ps --format "table {{.Names}}\t{{.Ports}}" 2>/dev/null | grep -E ":(8080|2345|5432|6379|4222|8222|3000|9090|9093|3100|9080|80|443)" || echo "  No Docker containers found on project ports"
    fi
}

# Function to suggest solutions
suggest_solutions() {
    echo -e "\n${BLUE}=== Troubleshooting Tips ===${NC}"
    echo -e "${YELLOW}If ports are in use:${NC}"
    echo "1. Stop Tiris services:    docker compose -f docker-compose.dev.yml down"
    echo "2. Kill specific processes: $0 --kill-dev"
    echo "3. Check what's using ports: $0 --detailed"
    echo "4. Change ports in config files if needed"
    echo ""
    echo -e "${YELLOW}Common port conflicts:${NC}"
    echo "• Port 8080: Other web servers (kill with: lsof -ti:8080 | xargs kill -9)"
    echo "• Port 5432: Other PostgreSQL instances"
    echo "• Port 6379: Other Redis instances"
    echo "• Port 3000: React dev servers, Grafana"
}

# Main execution
main() {
    echo -e "${BLUE}Tiris Backend - Port Availability Checker${NC}"
    echo -e "Checking ports required for the project...\n"
    
    case "${1:-}" in
        --dev|--development)
            check_port_group "Development Ports" "${DEV_PORTS[@]}"
            ;;
        --monitoring)
            check_port_group "Monitoring Ports" "${MONITORING_PORTS[@]}"
            ;;
        --prod|--production)
            check_port_group "Production Ports" "${PROD_PORTS[@]}"
            ;;
        --all)
            dev_result=0
            monitoring_result=0
            prod_result=0
            
            check_port_group "Development Ports" "${DEV_PORTS[@]}" || dev_result=1
            check_port_group "Monitoring Ports" "${MONITORING_PORTS[@]}" || monitoring_result=1
            check_port_group "Production Ports" "${PROD_PORTS[@]}" || prod_result=1
            
            echo -e "\n${BLUE}=== Overall Summary ===${NC}"
            [ $dev_result -eq 0 ] && echo -e "${GREEN}✓ Development: Ready${NC}" || echo -e "${RED}✗ Development: Port conflicts${NC}"
            [ $monitoring_result -eq 0 ] && echo -e "${GREEN}✓ Monitoring: Ready${NC}" || echo -e "${RED}✗ Monitoring: Port conflicts${NC}"
            [ $prod_result -eq 0 ] && echo -e "${GREEN}✓ Production: Ready${NC}" || echo -e "${RED}✗ Production: Port conflicts${NC}"
            ;;
        --detailed)
            show_detailed_usage
            ;;
        --kill-dev)
            echo -e "${YELLOW}Killing processes on development ports...${NC}"
            kill_port_processes "${DEV_PORTS[@]}"
            echo -e "\nRe-checking ports:"
            check_port_group "Development Ports" "${DEV_PORTS[@]}"
            ;;
        --kill-monitoring)
            echo -e "${YELLOW}Killing processes on monitoring ports...${NC}"
            kill_port_processes "${MONITORING_PORTS[@]}"
            ;;
        --help|-h)
            echo "Usage: $0 [OPTION]"
            echo ""
            echo "Options:"
            echo "  --dev, --development    Check development ports only"
            echo "  --monitoring           Check monitoring ports only"
            echo "  --prod, --production   Check production ports only"
            echo "  --all                  Check all ports (default)"
            echo "  --detailed             Show detailed port usage"
            echo "  --kill-dev             Kill processes on development ports"
            echo "  --kill-monitoring      Kill processes on monitoring ports"
            echo "  --help, -h             Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                     # Check all ports"
            echo "  $0 --dev               # Check only development ports"
            echo "  $0 --detailed          # Show what's using each port"
            echo "  $0 --kill-dev          # Free up development ports"
            ;;
        *)
            # Default: check development ports (most common use case)
            check_port_group "Development Ports" "${DEV_PORTS[@]}"
            result=$?
            
            if [ $result -ne 0 ]; then
                suggest_solutions
            fi
            
            echo -e "\n${YELLOW}Use '$0 --help' for more options${NC}"
            exit $result
            ;;
    esac
}

# Run main function with all arguments
main "$@"