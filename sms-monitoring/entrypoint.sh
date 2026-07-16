#!/bin/sh

# Route the virtual subnet to the simulator container
SIMULATOR_IP=$(getent hosts tasks.simulator | awk '{ print $1 }' | head -n 1)
if [ -z "$SIMULATOR_IP" ]; then
    SIMULATOR_IP=$(getent hosts simulator | awk '{ print $1 }' | head -n 1)
fi

if [ -n "$SIMULATOR_IP" ]; then
    echo "Adding route for 10.1.0.0/16 via $SIMULATOR_IP"
    ip route add 10.1.0.0/16 via $SIMULATOR_IP
else
    echo "Warning: Could not resolve sms_simulator IP"
fi

exec /app/monitoring-worker
