#!/usr/bin/env bash

set -eux

# This image tag corresponds to a Kubernetes version that kind installs using
# images at:
# https://hub.docker.com/r/kindest/node/tags
K8S_VERSION="v1.15.4"
KIND_IMAGE="quay.io/estroz/node:${K8S_VERSION}"
# TODO: use the below image once it is rebuilt with the following base image:
# kindest/base:v20190926-e6b6f7f0@sha256:1ec92b2910f2bfb8a4814cc90e15974a499f0c93d203b879277fa4bdb3762ce2
# KIND_IMAGE="docker.io/kindest/node:${K8S_VERSION}"

# Download the latest version of kind, which supports all versions of
# Kubernetes v1.11+.
# TODO: use "latest" release URL once full releases are being made.
# Currently at pre-release.
KIND_VERSION="v0.5.1"
curl -Lo kind https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-$(uname)-amd64
chmod +x kind
sudo mv kind /usr/local/bin/

# Create a cluster of version $K8S_VERSION.
kind create cluster --image="$KIND_IMAGE"

# Run this command externally after installation:
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

# kubectl is needed for the single namespace local test and the ansible tests.
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Use the "default" namespace, and ensure kind is running.
kubectl config set-context --current --namespace=default
kubectl cluster-info
