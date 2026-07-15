#!/bin/sh
set -e

IP_COUNT=${SIMULATOR_IP_COUNT:-10000}
SUBNET=${SIMULATOR_SUBNET:-10.1}
START_OCTET_3=${SIMULATOR_START_OCTET_3:-0}
START_OCTET_4=${SIMULATOR_START_OCTET_4:-1}

echo "=== Creating dummy0 interface ==="
ip link add dummy0 type dummy
ip link set dummy0 up

echo "=== Assigning ${IP_COUNT} IPs ==="
OCTET3=$START_OCTET_3
OCTET4=$START_OCTET_4
i=0
while [ $i -lt $IP_COUNT ]; do
    ip addr add ${SUBNET}.${OCTET3}.${OCTET4}/16 dev dummy0
    OCTET4=$((OCTET4 + 1))
    if [ $OCTET4 -gt 254 ]; then
        OCTET4=1
        OCTET3=$((OCTET3 + 1))
    fi
    i=$((i + 1))
done
echo "Assigned ${IP_COUNT} IPs to dummy0"

echo "=== Setting up nftables ==="
nft add table inet sim
nft add chain inet sim input { type filter hook input priority 0 \; }
nft add set inet sim down_ips { type ipv4_addr \; }
nft add rule inet sim input ip daddr @down_ips icmp type echo-request drop

echo "=== Starting API server ==="
exec ./server-simulator
