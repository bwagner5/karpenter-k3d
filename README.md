# k3d Karpenter Cloud Provider

The k3d Karpenter Cloud Provider is a demo of how to create a simple Cloud Provider for Karpenter.

You can also use this or your own minimal cloud provider to test our Karpenter Core without spinning up expensive instances in the cloud.

## What is k3d? 

The [k3d](https://github.com/k3d-io/k3d) project enables you to create a local K8s cluster on your laptop by running K8s nodes within docker containers.

## Getting Started

First, make sure you have `k3d` installed. You can find instructions here in the [k3d repo](https://github.com/k3d-io/k3d). 

Create a local k3d cluster:

```
k3d cluster create my-karpenter-cp
```

Query node and pods on the cluster to make sure everything came up:

```
kubectl get nodes
kubectl get pods -A
```

You can see the backing docker containers for the nodes:

```
docker ps
```

Now start the Karpenter k3d cloud provider:

```
## first install the Provisioner CRD from the Karpenter repo
kubectl apply -f <path-to-karpenter-repo>/charts/karpenter/crds/karpenter.sh_provisioners.yaml

K3D_HELPER_IMAGE_TAG='5.4.4' CLUSTER_ENDPOINT=https://localhost:60111 CLUSTER_NAME=my-karpenter-cp SYSTEM_NAMESPACE=kube-system go run cmd/main.go
```

Or install the Helm chart through the Makefile:

```
make apply
```

You can apply a sample Provisioner from the samples dir:

```
kubectl apply -f samples/provisioner.yaml
```

And now scale a deployment to see nodes get added!

```
kubectl scale deploy/metrics-server -n kube-system --replicas=3
```