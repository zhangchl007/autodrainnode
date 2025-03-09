Collecting workspace information# AutoDrainNode

AutoDrainNode is a Kubernetes controller that automatically drains nodes when they become unhealthy or are about to shut down, and uncordons them when they become ready again.

## Features

- Automatically detects when nodes become `NotReady` and drains them
- Watches for `Shutdown` events and proactively drains nodes
- Automatically uncordons nodes when they return to `Ready` state
- Respects DaemonSets by not evicting their pods
- Configurable timeout for pod eviction

## How It Works

The application:
1. Watches Kubernetes node status changes
2. Detects `NodeNotReady` or `Shutdown` events
3. Cordons the node to prevent new pods from being scheduled
4. Evicts all non-DaemonSet pods from the node
5. Waits for pod eviction to complete
6. Automatically uncordons nodes when they return to `Ready` state

## Prerequisites

- Kubernetes cluster
- RBAC permissions for:
  - Watching nodes and events
  - Updating node status
  - Evicting pods

## Installation

1. Create a service account with appropriate RBAC permissions
2. Deploy the application as a Deployment in your cluster

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: autodrainnode
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: autodrainnode
rules:
- apiGroups: [""]
  resources: ["nodes"]
  verbs: ["get", "list", "watch", "update"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "delete"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["daemonsets"]
  verbs: ["get", "list"]
- apiGroups: ["policy"]
  resources: ["poddisruptionbudgets", "evictions"]
  verbs: ["get", "list", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: autodrainnode
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: autodrainnode
subjects:
- kind: ServiceAccount
  name: autodrainnode
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: autodrainnode
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: autodrainnode
  template:
    metadata:
      labels:
        app: autodrainnode
    spec:
      serviceAccountName: autodrainnode
      containers:
      - name: autodrainnode
        image: zhangchl007/autodrainnode:latest
```

## Build and Deploy

1. Build the container image:

```sh
docker build -t zhangchl007/autodrainnode:latest .
docker push zhangchl007/autodrainnode:latest
```

2. Apply the YAML:

```sh
kubectl apply -f deployment.yaml
```

## Contributing

Feel free to open issues or pull requests for any bugs or feature requests.

## License

[Insert your license information here]