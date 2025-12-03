#!/bin/bash

# Prebid Server Local Run Script
# This script helps you easily start, stop, and manage Prebid Server locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PORT=8000
LOG_FILE="pbs_output.log"

# Function to print colored messages
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Function to check if PBS is running
is_running() {
    lsof -ti :$PORT > /dev/null 2>&1
}

# Function to get PBS process ID
get_pid() {
    lsof -ti :$PORT 2>/dev/null
}

# Function to start PBS
start_pbs() {
    if is_running; then
        print_warning "Prebid Server is already running on port $PORT (PID: $(get_pid))"
        echo "Use './run.sh stop' to stop it first, or './run.sh restart' to restart"
        exit 1
    fi

    print_info "Starting Prebid Server..."
    
    # Check if GDPR is configured
    if [ -z "$PBS_GDPR_DEFAULT_VALUE" ]; then
        print_warning "PBS_GDPR_DEFAULT_VALUE not set, using default: 0"
        export PBS_GDPR_DEFAULT_VALUE="0"
    fi

    # Check for MaxMind database and download if missing
    MAXMIND_DB="tmp/GeoLite2-Country.mmdb"
    mkdir -p tmp  # Ensure tmp directory exists
    if [ ! -f "$MAXMIND_DB" ]; then
        if [ -f ".env" ]; then
            print_info "MaxMind database not found, checking for credentials..."
            # Source .env to get MAXMIND_LICENSE_KEY
            set +e  # Temporarily disable exit on error for sourcing
            source .env 2>/dev/null
            set -e  # Re-enable exit on error
            
            # Export variables so they're available to subprocesses
            if [ -n "$MAXMIND_LICENSE_KEY" ]; then
                export MAXMIND_LICENSE_KEY
            fi
            if [ -n "$MAXMIND_ACCOUNT_ID" ]; then
                export MAXMIND_ACCOUNT_ID
            fi
            
            if [ -n "$MAXMIND_LICENSE_KEY" ]; then
                print_info "Downloading MaxMind database..."
                if ./scripts/download-maxmind.sh "$MAXMIND_DB"; then
                    print_success "MaxMind database downloaded successfully"
                else
                    print_warning "Failed to download MaxMind database, continuing without IP resolver"
                fi
            else
                print_warning "MAXMIND_LICENSE_KEY not found in .env, skipping database download"
                print_info "IP-based geo resolution will not be available"
            fi
        else
            print_warning ".env file not found, skipping MaxMind database download"
            print_info "IP-based geo resolution will not be available"
        fi
    else
        print_info "MaxMind database found at $MAXMIND_DB"
    fi

    # Build first for faster startup detection
    print_info "Building Prebid Server..."
    if ! go build -o prebid-server . 2>&1; then
        print_error "Build failed"
        exit 1
    fi
    
    # Start in background
    PBS_GDPR_DEFAULT_VALUE="$PBS_GDPR_DEFAULT_VALUE" ./prebid-server > "$LOG_FILE" 2>&1 &
    
    # Wait a bit for startup
    sleep 5
    
    # Check if it started successfully
    if is_running; then
        print_success "Prebid Server started successfully!"
        print_info "PID: $(get_pid)"
        print_info "Port: $PORT"
        print_info "Logs: $LOG_FILE"
        print_info "Status: http://localhost:$PORT/status"
        print_info "Auction: http://localhost:$PORT/openrtb2/auction"
        echo ""
        print_info "View logs: tail -f $LOG_FILE"
        print_info "Stop server: ./run.sh stop"
    else
        print_error "Failed to start Prebid Server"
        print_info "Check logs: tail -50 $LOG_FILE"
        exit 1
    fi
}

# Function to stop PBS
stop_pbs() {
    if ! is_running; then
        print_warning "Prebid Server is not running"
        exit 0
    fi

    local pid=$(get_pid)
    print_info "Stopping Prebid Server (PID: $pid)..."
    
    kill $pid 2>/dev/null || true
    
    # Wait for graceful shutdown
    local count=0
    while is_running && [ $count -lt 10 ]; do
        sleep 0.5
        count=$((count + 1))
    done
    
    # Force kill if still running
    if is_running; then
        print_warning "Forcing shutdown..."
        kill -9 $pid 2>/dev/null || true
        sleep 1
    fi
    
    if ! is_running; then
        print_success "Prebid Server stopped"
    else
        print_error "Failed to stop Prebid Server"
        exit 1
    fi
}

# Function to restart PBS
restart_pbs() {
    print_info "Restarting Prebid Server..."
    stop_pbs
    sleep 1
    start_pbs
}

# Function to show status
status_pbs() {
    if is_running; then
        local pid=$(get_pid)
        print_success "Prebid Server is running"
        echo "  PID: $pid"
        echo "  Port: $PORT"
        echo "  Logs: $LOG_FILE"
        echo ""
        
        # Test status endpoint
        print_info "Testing status endpoint..."
        local status_code=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT/status 2>/dev/null || echo "000")
        
        if [ "$status_code" = "204" ]; then
            print_success "Server is healthy (HTTP $status_code)"
        elif [ "$status_code" = "000" ]; then
            print_error "Cannot connect to server"
        else
            print_warning "Server responded with HTTP $status_code"
        fi
    else
        print_warning "Prebid Server is not running"
        echo "Use './run.sh start' to start it"
    fi
}

# Function to show logs
logs_pbs() {
    if [ ! -f "$LOG_FILE" ]; then
        print_error "Log file not found: $LOG_FILE"
        exit 1
    fi
    
    local lines=${1:-50}
    print_info "Showing last $lines lines of logs (Ctrl+C to exit live mode)..."
    echo ""
    
    if [ "$2" = "follow" ]; then
        tail -n $lines -f "$LOG_FILE"
    else
        tail -n $lines "$LOG_FILE"
    fi
}

# Function to test auction
test_auction() {
    if ! is_running; then
        print_error "Prebid Server is not running. Start it with './run.sh start'"
        exit 1
    fi
    
    print_info "Testing auction endpoint..."
    
    if [ -f "test_auction.json" ]; then
        print_info "Using test_auction.json..."
        local response=$(curl -s -X POST http://localhost:$PORT/openrtb2/auction \
            -H "Content-Type: application/json" \
            -d @test_auction.json)
    else
        print_info "Using inline test request..."
        local response=$(curl -s -X POST http://localhost:$PORT/openrtb2/auction \
            -H "Content-Type: application/json" \
            -d '{
                "id": "test-'$(date +%s)'",
                "imp": [{
                    "id": "imp-1",
                    "banner": {"format": [{"w": 300, "h": 250}]}
                }],
                "site": {
                    "domain": "example.com",
                    "page": "http://example.com/test"
                },
                "device": {"ua": "Mozilla/5.0", "ip": "127.0.0.1"},
                "at": 1,
                "cur": ["USD"]
            }')
    fi
    
    if [ -n "$response" ]; then
        print_success "Auction response received:"
        echo "$response" | python3 -m json.tool 2>/dev/null || echo "$response"
    else
        print_error "No response from auction endpoint"
        exit 1
    fi
}

# Function to show usage
usage() {
    echo "Prebid Server Local Management Script"
    echo ""
    echo "Usage: ./run.sh [command] [options]"
    echo ""
    echo "Commands:"
    echo "  start           Start Prebid Server in background"
    echo "  stop            Stop Prebid Server"
    echo "  restart         Restart Prebid Server"
    echo "  status          Show server status and health"
    echo "  logs [N]        Show last N lines of logs (default: 50)"
    echo "  logs [N] follow Show last N lines and follow new logs"
    echo "  test            Run a test auction"
    echo "  help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  ./run.sh start                # Start PBS"
    echo "  ./run.sh status               # Check if running"
    echo "  ./run.sh logs 100             # Show last 100 log lines"
    echo "  ./run.sh logs 50 follow       # Tail logs in real-time"
    echo "  ./run.sh test                 # Test auction endpoint"
    echo "  ./run.sh restart              # Restart PBS"
    echo "  ./run.sh stop                 # Stop PBS"
    echo ""
    echo "Environment Variables:"
    echo "  PBS_GDPR_DEFAULT_VALUE        GDPR default (0 or 1, default: 0)"
    echo ""
    echo "Quick Start:"
    echo "  1. ./run.sh start             # Start the server"
    echo "  2. ./run.sh test              # Test an auction"
    echo "  3. ./run.sh logs 20 follow    # Watch logs"
    echo "  4. ./run.sh stop              # Stop when done"
}

# Main script logic
case "${1:-help}" in
    start)
        start_pbs
        ;;
    stop)
        stop_pbs
        ;;
    restart)
        restart_pbs
        ;;
    status)
        status_pbs
        ;;
    logs)
        logs_pbs "${2:-50}" "$3"
        ;;
    test)
        test_auction
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        usage
        exit 1
        ;;
esac

