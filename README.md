
# Rbati CNI 🐉

![Build Status](https://github.com/chahid001/rbati-cni/actions/workflows/release.yml/badge.svg)
[![Latest Release](https://img.shields.io/github/v/release/chahid001/rbati-cni?display_name=tag&sort=semver)](https://github.com/chahid001/rbati-cni/releases/latest)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

A simple, yet powerful Container Network Interface (CNI) for Kubernetes, built from scratch in Go. Rbati CNI provides a robust L2 overlay network using VXLAN, enabling seamless pod-to-pod communication across different nodes.

---
## ✨ Features

* **VXLAN Overlay**: Creates a scalable Layer 2 network over your existing Layer 3 infrastructure.
* **IPAM with embedded bolt DB**: Manages IP address allocation for pods.
* **Controller/Agent Model**: Uses Kubernetes informers to react instantly to cluster changes (new nodes, pods, etc.).
* **NAT for Internet Access**: Provides outbound internet connectivity for pods using `iptables` masquerading.
* **No External Dependencies**: Doesn't require any external key-value stores like etcd.
* **Lightweight & Performant**: Written in Go for minimal overhead.

---
## 🏗️ Architecture
Rbati CNI uses a standard two-component architecture:
1.  **`agent-controller` (DaemonSet)**: A long-running process that runs on every node in the cluster.
    * Creates and manages the VXLAN and bridge interfaces.
    * Sets up `iptables` for NAT.
    * Watches the Kubernetes API for new nodes and pods to update routing and forwarding tables.

2.  **`rbati` CNI Plugin (Binary)**: A short-lived executable located in `/opt/cni/bin/`.
    * Called by the container runtime (like containerd) during pod creation (`ADD`) and deletion (`DEL`).
    * Handles IPAM, creating the veth pair for the pod, and connecting it to the CNI bridge.

 ```
 ┌────────────────────────── Node 1 ──────────────────────────┐   ┌────────────────────────── Node 2 ──────────────────────────┐
 │                                                            │   │                                                            │
 │  ┌─────────┐    ┌─────────┐                                │   │                             ┌─────────┐    ┌─────────┐     │
 │  │ Pod A   │    │ Pod B   │                                │   │                             │ Pod C   │    │ Pod D   │     │
 │  └───┬─────┘    └───┬─────┘                                │   │                             └───┬─────┘    └───┬─────┘     │
 │      │ veth         │ veth                                 │   │                                 │ veth         │ veth      │
 │      └─────┬────────┘                                      │   │                                 └─────┬────────┘           │
 │            │                                               │   │                                       │                    │
 │      ┌─────▼─────┐                                         │   │                                 ┌─────▼─────┐              │
 │      │ rbati.0   │                                         │   │                                 │ rbati.0   │              │
 │      │ (Bridge)  │                                         │   │                                 │ (Bridge)  │              │
 │      └─────┬─────┘                                         │   │                                 └─────┬─────┘              │
 │            │                                               │   │                                       │                    │
 │      ┌─────▼─────┐       Cross-Node Pod Communication      │   │    Cross-Node Pod Communication ┌─────▼─────┐              │
 │      │ rbati.1   ├<═════════════════════════════════════════════════════════════════════════════>┤ rbati.1   │              │
 │      │ (VXLAN)   │                                         │   │                                 │ (VXLAN)   │              │
 │      └─────┬─────┘                                         │   │                                 └─────┬─────┘              │
 │            │                                               │   │                                       │                    │
 │      ┌─────▼─────┐                                         │   │                                 ┌─────▼─────┐              │
 │      │   eth0    ├<─────────────────────────────────── Underlay Network ────────────────────────>┤   eth0    │              │
 │      └───────────┘                                         │   │                                 └───────────┘              │
 │                                                            │   │                                                            │
 └────────────────────────────────────────────────────────────┘   └────────────────────────────────────────────────────────────┘
 ```

---
## 🚀 Getting Started
### Prerequisites
* A Kubernetes cluster (v1.20+ recommended).
* `sudo` access on all nodes to install the CNI binary.
### Installation
The recommended way to install Rbati CNI is by using the assets from the latest GitHub Release.
1.  **Download the latest release**
    Go to the [**Releases Page**](https://github.com/your-username/your-cni-repo/releases/latest) and download the `rbati-cni.tar.gz` archive.
2.  **Install on Each Node**
    Copy the archive to each node in your cluster, extract it, and run the installation script. This will copy the CNI binary and configuration to the correct locations and apply the RBAC and deamonset manifest.
    ```bash
    tar -xvf rbati-cni.tar.gz
    cd rbati-cni
    sudo ./install.sh 
    ```

    ```bash
    tar -xvf rbati-cni.tar.gz
    cd rbati-cni
    sudo ./worker.sh 
    ```

Your CNI is now active! New pods will automatically use Rbati CNI for networking.
---
## 🔧 Configuration

The CNI configuration is managed via a file in `/etc/cni/net.d/`, typically named `10-rbati.conf`. The installer places a default version for you.
```json
{
  "cniVersion": "0.4.0",
  "name": "rbati-cni",
  "type": "rbati",
  "iface": "eth1",
  "bridge": "rbati.0"
}
```
**PS**: Please change the iface, to the default one of your system on which the VXLAN can based on for multi node networking.
---
## 💻 Development Environment
I did use K3S for my developement environement
```bash
curl -sfL https://get.k3s.io | \
  INSTALL_K3S_EXEC="\
    --bind-address=<host IP> \
    --node-ip=<host IP> \
    --flannel-backend=none \
    --disable-network-policy" \
  K3S_KUBECONFIG_MODE="644" \
  K3S_NODE_NAME=schahidS \
  sh -
```

## 💻 Development
Want to contribute or build from source?
1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/chahid001/rbati-cni.git](https://github.com/chahid001/rbati-cni.git)
    cd rbati-cni
    ```
2.  **Build the binaries:**
    ```bash
    make buildDev
    ```
---
## 📄 License
This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.