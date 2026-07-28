#!/bin/sh

# Route the virtual subnet to the simulator container
# Wait for simulator to be ready (up to 30 seconds)
MAX_RETRIES=30
RETRY_COUNT=0
SIMULATOR_IP=""

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    SIMULATOR_IP=$(ping -c 1 tasks.simulator 2>/dev/null | awk -F'[()]' '/PING/ { print $2 }')
    if [ -z "$SIMULATOR_IP" ]; then
        SIMULATOR_IP=$(ping -c 1 simulator 2>/dev/null | awk -F'[()]' '/PING/ { print $2 }')
    fi
    
    if [ -n "$SIMULATOR_IP" ]; then
        break
    fi
    
    echo "Waiting for simulator DNS to resolve... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 1
    RETRY_COUNT=$((RETRY_COUNT+1))
done

if [ -n "$SIMULATOR_IP" ]; then
    echo "Adding route for 10.1.0.0/16 via $SIMULATOR_IP"
    ip route add 10.1.0.0/16 via $SIMULATOR_IP
else
    echo "Error: Could not resolve simulator IP after 30 seconds"
    exit 1
fi

exec /app/monitoring-worker
