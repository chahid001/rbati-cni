#!/bin/bash

set -e

PLUGIN_NAME="rbati-cni"
PLUGIN_BIN="/schahid/$PLUGIN_NAME"

echo "[INFO] Installing $PLUGIN_NAME on worker..."

# Install the CNI binary
sudo mkdir -p /opt/cni/bin
sudo cp "$PLUGIN_BIN" /opt/cni/bin/
sudo chmod +x /opt/cni/bin/$PLUGIN_NAME

# Install loopback plugin from K3s
sudo cp /var/lib/rancher/k3s/data/*/bin/loopback /opt/cni/bin/ || {
  echo "[WARN] loopback plugin not found in K3s path"
}
sudo chmod +x /opt/cni/bin/loopback

# Create the CNI config with unique subnet
sudo mkdir -p /etc/cni/net.d
sudo tee /etc/cni/net.d/10-rbati.conf > /dev/null <<EOF
{
  "cniVersion": "0.4.0",
  "name": "rbati",
  "type": "rbati-cni",
  "bridge": "cni0",
  "bridgeIP": "172.16.2.1",
  "iface": "eth1",
  "ipam": {
    "subnet": "172.16.2.0/24",
    "routes": [
      {
        "dst": "0.0.0.0/0",
        "gw": "172.16.2.1"
      }
    ],
    "dns": {
      "nameservers": ["8.8.8.8", "1.1.1.1"],
      "search": ["svc.cluster.local"]
    }
  }
}
EOF

echo "[INFO] Worker installation complete."
