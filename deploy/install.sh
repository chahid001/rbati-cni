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
sudo cp /var/lib/rancher/k3s/data/*/bin/loopback /opt/cni/bin/ || echo "[WARN] Loopback plugin not found"
sudo chmod +x /opt/cni/bin/loopback || true

# Create the CNI config
sudo mkdir -p /etc/cni/net.d
sudo tee /etc/cni/net.d/10-rbati.conf > /dev/null <<EOF
{
  "cniVersion": "0.4.0",
  "name": "rbati",
  "type": "rbati-cni",     
  "iface": "eth1",
  "controllerIP": "192.168.56.110",
  "ipam": {           
    "subnet": "172.16.1.0/24",
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
  namespace: kube-system
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
  namespace: kube-system
EOF

echo "[INFO] Deploying agent (DaemonSet)..."

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: rbati-agent
  namespace: kube-system
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
      dnsPolicy: ClusterFirstWithHostNet
      containers:
      - name: agent
        image: schahid01/rbati-agent:latest
        securityContext:
          privileged: true
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: cniconf
          mountPath: /etc/cni/net.d
        - name: cni-bin
          mountPath: /opt/cni/bin
      volumes:
      - name: cniconf
        hostPath:
          path: /etc/cni/net.d
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
EOF

echo "[INFO] Deploying controller (Deployment + Service)..."

kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: rbati-controller
  namespace: kube-system
spec:
  selector:
    app: rbati-controller
  ports:
  - name: grpc
    port: 50051
    targetPort: 50051
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rbati-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rbati-controller
  template:
    metadata:
      labels:
        app: rbati-controller
    spec:
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
      serviceAccountName: rbati-agent
      containers:
      - name: controller
        image: schahid01/rbati-controller:latest
        securityContext:
          privileged: true
        ports:
        - containerPort: 50051
        volumeMounts:
        - name: cniconf
          mountPath: /etc/cni/net.d
        - name: cni-bin
          mountPath: /opt/cni/bin
        - name: ipam-db
          mountPath: /var/lib/cni
      volumes:
      - name: cniconf
        hostPath:
          path: /etc/cni/net.d
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
      - name: ipam-db
        hostPath:
          path: /var/lib/rbati-cni
          type: DirectoryOrCreate
EOF

echo "[DONE] Installation complete."
