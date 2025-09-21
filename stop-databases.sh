#!/bin/bash

# Database Instance Management Script
# This script helps manage both local and Docker PostgreSQL instances

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a process is running on a port
check_port() {
    local port=$1
    if lsof -i :$port > /dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Function to stop local PostgreSQL (multiple methods)
stop_local_postgres() {
    print_warning "Skipping local PostgreSQL - we don't manage the default port 5432"
    print_status "This script only manages Docker containers on custom ports"
    print_status "Your local PostgreSQL on port 5432 will remain untouched"
}

# Function to stop Docker containers
stop_docker_containers() {
    print_status "Stopping Docker containers..."
    
    if command -v docker > /dev/null 2>&1; then
        # Stop our specific containers
        local containers=("database-mcp-postgres" "database-mcp-clickhouse")
        
        for container in "${containers[@]}"; do
            if docker ps -q -f name="$container" | grep -q .; then
                print_status "Stopping container: $container"
                docker stop "$container" || print_warning "Failed to stop $container"
            else
                print_status "Container $container not running"
            fi
        done
        
        # Stop via docker-compose if available
        if [[ -f "docker-compose.yml" ]]; then
            print_status "Stopping Docker Compose services..."
            docker compose down 2>/dev/null || docker-compose down 2>/dev/null || true
        fi
        
        print_success "Docker containers stopped"
    else
        print_warning "Docker not found"
    fi
}

# Function to check port status
check_ports() {
    print_status "Checking port status (Docker containers only)..."
    
    # Only check our Docker ports, not the default PostgreSQL port
    local ports=(5433 9001 8124)
    local port_descriptions=("PostgreSQL (Docker)" "ClickHouse Native (Docker)" "ClickHouse HTTP (Docker)")
    
    for i in "${!ports[@]}"; do
        local port=${ports[$i]}
        local desc=${port_descriptions[$i]}
        
        if check_port $port; then
            local process=$(lsof -i :$port | tail -n 1 | awk '{print $1}')
            print_warning "Port $port ($desc) is in use by: $process"
        else
            print_success "Port $port ($desc) is free"
        fi
    done
    
    # Show local PostgreSQL status without managing it
    if check_port 5432; then
        print_status "Port 5432 (Local PostgreSQL) - in use (not managed by this script)"
    else
        print_status "Port 5432 (Local PostgreSQL) - free"
    fi
}

# Function to show usage
show_usage() {
    echo "Database Instance Management Script"
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  stop-all        Stop all Docker containers (leaves local PostgreSQL alone)"
    echo "  stop-local      Show warning about not managing local PostgreSQL"
    echo "  stop-docker     Stop only Docker containers"
    echo "  check-ports     Check status of Docker container ports"
    echo "  status          Show current status of all instances"
    echo "  help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 stop-all     # Stop Docker containers only"
    echo "  $0 status       # Check what's running"
    echo "  $0 check-ports  # See which Docker ports are in use"
    echo ""
    echo "Note: This script does NOT manage your local PostgreSQL on port 5432"
}

# Function to show status
show_status() {
    print_status "Database Instance Status Report"
    echo ""
    
    print_status "=== Local PostgreSQL (Not Managed) ==="
    if check_port 5432; then
        print_status "Local PostgreSQL RUNNING on port 5432 (your local instance - not managed)"
    else
        print_status "Local PostgreSQL NOT DETECTED on port 5432"
    fi
    
    print_status "=== Docker Containers (Managed by this script) ==="
    if command -v docker > /dev/null 2>&1; then
        local containers=("database-mcp-postgres" "database-mcp-clickhouse")
        
        for container in "${containers[@]}"; do
            if docker ps -q -f name="$container" | grep -q .; then
                local status=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep "$container" | awk '{print $2, $3}')
                print_success "$container RUNNING ($status)"
            else
                print_status "$container STOPPED"
            fi
        done
    else
        print_warning "Docker not available"
    fi
    
    echo ""
    check_ports
}

# Main script logic
case "${1:-status}" in
    "stop-all")
        print_status "Stopping Docker database containers only..."
        print_status "Your local PostgreSQL on port 5432 will remain untouched"
        stop_docker_containers
        echo ""
        check_ports
        ;;
    "stop-local")
        stop_local_postgres
        ;;
    "stop-docker")
        stop_docker_containers
        echo ""
        check_ports
        ;;
    "check-ports")
        check_ports
        ;;
    "status")
        show_status
        ;;
    "help"|"-h"|"--help")
        show_usage
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_usage
        exit 1
        ;;
esac