#!/bin/bash
set -euo pipefail

PLUGIN_NAME="rbati-d"
PLUGIN_BIN="./bin/$PLUGIN_NAME"

log_info()  { echo -e "\033[1;34m[INFO]\033[0m $*"; }
log_warn()  { echo -e "\033[1;33m[WARN]\033[0m $*"; }

log_info "Installing $PLUGIN_NAME..."

sudo install -m 755 -D "$PLUGIN_BIN" /opt/cni/bin/$PLUGIN_NAME

sudo cp ./bin/loopback /opt/cni/bin/
sudo chmod +x /opt/cni/bin/loopback


sudo mkdir -p /etc/cni/net.d
sudo cp 10-rbati.conf /etc/cni/net.d/

log_info "Applying RBAC permissions for the agent..."

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
  name: rbati-agent-controller
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "patch", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: rbati-agent-controller-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: rbati-agent-controller
subjects:
- kind: ServiceAccount
  name: rbati-agent
  namespace: kube-system
EOF

log_info "Deploying rbati-agent DaemonSet..."

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
        - name: rbati-env
          mountPath: /run/rbati-cni
      volumes:
      - name: cniconf
        hostPath:
          path: /etc/cni/net.d
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
      - name: rbati-env
        hostPath:
          path: /run/rbati-cni
          type: DirectoryOrCreate
EOF

log_info "Control plane setup complete."
