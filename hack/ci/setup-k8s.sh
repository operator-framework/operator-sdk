#!/usr/bin/env bash

set -eux

# This image tag corresponds to a Kubernetes version that kind installs using
# images at:
# https://hub.docker.com/r/kindest/node/tags
K8S_VERSION="v1.16.3"
KIND_IMAGE="docker.io/kindest/node:${K8S_VERSION}"

# Download the latest version of kind, which supports all versions of
# Kubernetes v1.11+.
curl -Lo kind https://github.com/kubernetes-sigs/kind/releases/latest/download/kind-$(uname)-amd64
chmod +x kind
sudo mv kind /usr/local/bin/

# Create a cluster of version $K8S_VERSION.
kind create cluster --image="$KIND_IMAGE"

# Run this command externally after installation:
kind export kubeconfig

# kubectl is needed for the single namespace local test and the ansible tests.
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Use the "default" namespace, and ensure kind is running.
kubectl config set-context --current --namespace=default
kubectl cluster-info
