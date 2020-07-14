#!/usr/bin/env bash

set -eu

source hack/lib/common.sh

# This image tag corresponds to a Kubernetes version that kind installs using node images:
# https://hub.docker.com/r/kindest/node/tags
K8S_VERSION=$1
KIND_IMAGE="docker.io/kindest/node:${K8S_VERSION}"

# Download the latest version of kind and kubectl, which are needed for project unit and e2e tests.
prepare_staging_dir $tmp_sdk_root
install_kind $tmp_sdk_root
install_kubectl $tmp_sdk_root
fetch_tools $tmp_sdk_root
setup_envs $tmp_sdk_root

# Create a cluster of version $K8S_VERSION.
kind create cluster --image="$KIND_IMAGE"

# Run this command externally after installation:
kind export kubeconfig

# Use the "default" namespace, and ensure kind is running.
kubectl config set-context --current --namespace=default
kubectl cluster-info
