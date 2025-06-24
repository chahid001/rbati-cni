#!/bin/bash

set -e

PLUGIN_NAME="rbati-cni"
PLUGIN_BIN="./bin/$PLUGIN_NAME"

echo "[INFO] Installing $PLUGIN_NAME..."

# Install the CNI binary
sudo mkdir -p /opt/cni/bin
sudo cp "$PLUGIN_BIN" /opt/cni/bin/
sudo chmod +x /opt/cni/bin/$PLUGIN_NAME

# Install loopback plugin from K3s
sudo cp /var/lib/rancher/k3s/data/f10c229b8445ff9999c26495950a5c8b2e08fd13f1889d38f994edeebf026236/bin/loopback /opt/cni/bin/
sudo chmod +x /opt/cni/bin/loopback

# Create the CNI config
sudo mkdir -p /etc/cni/net.d
sudo tee /etc/cni/net.d/10-rbati.conf > /dev/null <<EOF
{
  "cniVersion": "0.4.0",
  "name": "rbati",
  "type": "rbati-cni",     
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
      "search": ["svc.cluster.local"]
    }
  }
}
EOF
echo "[INFO] Creating RBAC for node access..."

kubectl apply -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rbati-agent
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: rbati-agent-node-reader
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: rbati-agent-node-reader-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: rbati-agent-node-reader
subjects:
- kind: ServiceAccount
  name: rbati-agent
  namespace: default
EOF


echo "[INFO] Applying controller DaemonSet..."

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: rbati-agent
  labels:
    app: rbati-agent
spec:
  selector:
    matchLabels:
      app: rbati-agent
  template:
    metadata:
      labels:
        app: rbati-agent
    spec:
      hostNetwork: true
      serviceAccountName: rbati-agent
      containers:
      - name: agent
        image: schahid01/rbati-agent:latest
        securityContext:
          privileged: true  
        volumeMounts:
        - name: cniconf
          mountPath: /etc/cni/net.d
        - name: cni-bin
          mountPath: /opt/cni/bin
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
      volumes:
      - name: cniconf
        hostPath:
          path: /etc/cni/net.d
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
EOF

echo "[DONE] Installation complete."
