#!/bin/bash
echo "[INFO] Cleaning up VXLAN and bridge..."

ip link del rbati.1 2>/dev/null || echo "VXLAN not found"
ip link del cni0 2>/dev/null || echo "Bridge not found"

rm -f /etc/cni/net.d/10-rbati.conflist
rm -f /opt/cni/bin/rbati-cni

echo "[INFO] Cleanup complete."
