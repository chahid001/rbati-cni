#!/bin/bash
set -euo pipefail

PLUGIN_NAME="rbati-d"
PLUGIN_BIN="/schahid/$PLUGIN_NAME"

log_info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

log_info "Installing $PLUGIN_NAME on worker..."

# Install the CNI plugin binary
sudo install -m 755 -D "$PLUGIN_BIN" /opt/cni/bin/$PLUGIN_NAME

# Try to install the loopback plugin from K3s
if ! sudo cp /var/lib/rancher/k3s/data/*/bin/loopback /opt/cni/bin/ 2>/dev/null; then
  log_warn "Loopback plugin not found in K3s path"
else
  sudo chmod +x /opt/cni/bin/loopback
fi

# Apply CNI configuration
sudo mkdir -p /etc/cni/net.d
sudo cp 10-rbati.conf /etc/cni/net.d/

log_info "Worker installation complete."

