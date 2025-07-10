#!/bin/bash
echo "[INFO] Cleaning up VXLAN and bridge..."

sudo ip link del rbati.1 2>/dev/null || echo "VXLAN not found"
sudo ip link del rbati.0 2>/dev/null || echo "Bridge not found"

sudo rm -f /etc/cni/net.d/10-rbati.conflist
sudo rm -f /opt/cni/bin/rbati-cni
sudo rm -rf /var/lib/cni/rbati-cni
sudo rm -rf /tmp/mycni.log
sudo rm -rf /run/rbati-cni


echo "[INFO] Cleanup complete."
