#!/bin/bash
set -euo pipefail

log_info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

log_info "Cleaning up VXLAN interfaces and CNI files..."

sudo ip link del rbati.1 2>/dev/null || log_warn "VXLAN interface 'rbati.1' not found"
sudo ip link del rbati.0 2>/dev/null || log_warn "Bridge interface 'rbati.0' not found"

sudo rm -f /etc/cni/net.d/10-rbati.conf 2>/dev/null || true
sudo rm -f /opt/cni/bin/rbati-cni 2>/dev/null || true
sudo rm -rf /var/lib/cni/rbati-cni /tmp/mycni.log /run/rbati-cni 2>/dev/null || true

log_info "Cleanup complete."
