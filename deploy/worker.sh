#!/bin/bash
set -euo pipefail

PLUGIN_NAME="rbati-d"
PLUGIN_BIN="./bin/$PLUGIN_NAME"

log_info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

log_info "Installing $PLUGIN_NAME on worker..."

# Install the CNI plugin binary
sudo install -m 755 -D "$PLUGIN_BIN" /opt/cni/bin/$PLUGIN_NAME

log_info "Downloading Loopback binary..."
curl -LO https://github.com/containernetworking/plugins/releases/download/v1.4.0/cni-plugins-linux-amd64-v1.4.0.tgz
tar -xvzf cni-plugins-linux-amd64-v1.4.0.tgz ./loopback

log_info "Cleaning up..."
sudo rm -rf cni-plugins-linux-amd64-v1.4.0.tgz

sudo cp loopback /opt/cni/bin/
sudo chmod +x /opt/cni/bin/loopback
# Apply CNI configuration
sudo mkdir -p /etc/cni/net.d
sudo cp 10-rbati.conf /etc/cni/net.d/

log_info "Worker installation complete."

