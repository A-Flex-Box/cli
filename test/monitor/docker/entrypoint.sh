#!/bin/bash
set -e

echo "=== CLI Monitor Test Environment ==="
echo "Started at: $(date)"
echo ""

# Create output directory
mkdir -p /output /recordings

# Function to log output
log_output() {
    local name=$1
    local output_file="/output/${name}_$(date +%Y%m%d_%H%M%S).log"
    tee "$output_file"
    echo ""
    echo "Output saved to: $output_file"
}

# Run the requested command
case "$1" in
    monitor)
        echo "Starting monitor with recording..."
        export ASCIINEMA_REC=1
        
        # Start asciinema recording
        asciinema rec -q /recordings/monitor_$(date +%Y%m%d_%H%M%S).cast &
        ASCIINEMA_PID=$!
        
        # Run monitor
        timeout 10s cli monitor --no-capture 2>&1 | log_output "monitor" || true
        
        # Stop recording
        kill $ASCIINEMA_PID 2>/dev/null || true
        ;;
        
    test-keys)
        echo "Testing keyboard responsiveness..."
        echo "This tests Tab, Arrow keys, and q for quit"
        
        # Use expect-like testing
        {
            sleep 1
            echo "Tab pressed"
            sleep 0.5
            echo "Testing right arrow"
            sleep 0.5
            echo "Testing left arrow"
            sleep 0.5
            echo "Testing 'r' for refresh"
            sleep 0.5
            echo "Testing 'q' to quit"
        } | timeout 5s cli monitor --no-capture 2>&1 | log_output "test-keys"
        ;;
        
    bandwidth)
        echo "Testing bandwidth monitoring..."
        timeout 5s cli monitor --no-capture 2>&1 | log_output "bandwidth" || true
        ;;
        
    dns)
        echo "Testing DNS monitoring..."
        # Generate some DNS traffic
        nslookup google.com &
        nslookup github.com &
        timeout 5s cli monitor --no-capture 2>&1 | log_output "dns" || true
        ;;
        
    help)
        cli monitor --help | log_output "help"
        ;;
        
    *)
        echo "Usage: $0 {monitor|test-keys|bandwidth|dns|help}"
        echo ""
        echo "Running default: monitor"
        timeout 10s cli monitor --no-capture 2>&1 | log_output "monitor" || true
        ;;
esac

echo ""
echo "=== Test Complete ==="
echo "Outputs saved to /output/"
echo "Recordings saved to /recordings/"
ls -la /output/ /recordings/ 2>/dev/null || true