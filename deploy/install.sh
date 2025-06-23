#!/bin/bash

set -e

PLUGIN_NAME="rbati-cni"
PLUGIN_BIN="./bin/$PLUGIN_NAME"

echo "[INFO] Installing $PLUGIN_NAME..."

sudo mkdir -p /opt/cni/bin
sudo cp "$PLUGIN_BIN" /opt/cni/bin/
sudo chmod +x /opt/cni/bin/$PLUGIN_NAME

sudo cp /var/lib/rancher/k3s/data/f10c229b8445ff9999c26495950a5c8b2e08fd13f1889d38f994edeebf026236/bin/loopback /opt/cni/bin/
sudo chmod +x /opt/cni/bin/loopback


sudo mkdir -p /etc/cni/net.d
sudo tee /etc/cni/net.d/10-ftcni.conflist > /dev/null <<EOF
{
  "cniVersion": "0.4.0",
  "name": "ftcni",
  "plugins": [
    {
      "type": "ft_cni",
      "bridge": "cni0",
      "bridgeIP": "172.16.1.1",
      "iface": "eth1",
      "ipam": {
        "subnet": "172.16.1.0/16",
        "routes": [
	        {
		        "dst": "0.0.0.0/0",
		        "gw": "172.16.1.1"
		      }
	      ],
	      "dns": {
	        "nameservers": ["8.8.8.8", "1.1.1.1"],
	        "search": ["svc.cluster.local"],
	      }
      }
    },
    {
      "type": "loopback"
    }
  ]
}
EOF

echo "[INFO] Applying controller DaemonSet..."

kubectl apply -f ./deploy/rbati-agent.yaml

echo "[DONE] Installation complete."
