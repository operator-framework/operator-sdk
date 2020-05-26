#!/usr/bin/env bash

# Skip fetching and untaring the tools by setting the SKIP_FETCH_TOOLS variable
# in your environment to any value:
#
# $ SKIP_FETCH_TOOLS=1 ./test.sh
#
# If you skip fetching tools, this script will use the tools already on your
# machine, but rebuild the operator-sdk binary.
SKIP_FETCH_TOOLS=${SKIP_FETCH_TOOLS:-""}
# Current version of the 'kind' binary. Update this when a new breaking release
# is made for a docker.io/kindest/node:${K8S_VERSION} image.
KIND_VERSION="v0.8.1"
# ENVTEST_TOOLS_VERSION is the version of k8s server tarballs used for envtest.
# TODO: use K8S_VERSION once we start building our own server binary tarballs.
ENVTEST_TOOLS_VERSION="1.16.4"
# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:
NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header_color=$'\e[1;33m'
  error_color=$'\e[0;31m'
  reset_color=$'\e[0m'
else
  header_color=''
  error_color=''
  reset_color=''
fi

# Roots used by tests.
tmp_root=/tmp
tmp_sdk_root=$tmp_root/operator-sdk

function log() { printf '%s\n' "$*"; }
function error() { error_text "ERROR:" $* >&2; }
function fatal() { error "$@"; exit 1; }

function header_text {
  echo "$header_color$*$reset_color"
}

function error_text {
  echo "$error_color$*$reset_color"
}

function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}

# Install the ServiceMonitor CustomResourceDefinition so tests can verify that
# the ServiceMonitor resource is created for the operator.
function install_service_monitor_crd {
  kubectl apply -f https://raw.githubusercontent.com/coreos/prometheus-operator/release-0.35/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml
}

# prepare the e2e test staging dir, containing test tools (SKIP_FETCH_TOOLS aware).
function prepare_staging_dir {

  header_text "preparing staging dir $1"

  if [[ -z "$SKIP_FETCH_TOOLS" ]]; then
    rm -rf "$1"
  else
    rm -f "$1/bin/operator-sdk"
  fi

  mkdir -p "$1"
}

# Fetch k8s API gen tools and make it available under $1/bin.
function fetch_tools {
  if [[ -z "$SKIP_FETCH_TOOLS" ]]; then
    fetch_envtest_tools $@
    install_kind $@
  fi
}

# Fetch tools required for envtest.
function fetch_envtest_tools {

  # TODO: make our own tarball containing envtest binaries: etcd, kubectl, kube-apiserver
  #
  # To get k8s server binaries:
  # server_tar="kubernetes-server-$(go env GOOS)-$(go env GOARCH).tar.gz"
  # url=https://dl.k8s.io/$K8S_VERSION/$server_tar
  # curl -fL --retry 3 --keepalive-time 2 "${url}" -o "${tmp_sdk_root}/${server_tar}"
  # tar -zxvf "${tmp_sdk_root}/${server_tar}"

  local tools_archive_name="kubebuilder-tools-${ENVTEST_TOOLS_VERSION}-$(go env GOOS)-$(go env GOARCH).tar.gz"
  local tools_download_url="https://storage.googleapis.com/kubebuilder-tools/$tools_archive_name"

  local tools_archive_path="$1/$tools_archive_name"
  if [[ ! -f $tools_archive_path ]]; then
    header_text "fetching envtest tools"
    curl -sSLo "$tools_archive_path" $tools_download_url
  else
    header_text "using existing envtest tools in $tools_archive_path"
  fi
  tar -zvxf "$tools_archive_path" -C "$1/" --strip-components=1
}

# Set up test and envtest vars
function setup_envs {
  header_text "setting up env vars"

  export PATH="$1"/bin:$PATH
  export TEST_ASSET_KUBECTL="$1"/bin/kubectl
  export TEST_ASSET_KUBE_APISERVER="$1"/bin/kube-apiserver
  export TEST_ASSET_ETCD="$1"/bin/etcd
}

# Build the operator-sdk binary.
function build_sdk {
  header_text "building operator-sdk"

  GO111MODULE=on make build/operator-sdk
  mv ./build/operator-sdk "$1"/bin/operator-sdk
}

# Install the 'kind' binary at version $KIND_VERSION.
function install_kind {

  local kind_path="${1}/bin/kind"

  header_text "installing kind $KIND_VERSION"
  local kind_binary="kind-$(go env GOOS)-$(go env GOARCH)"
  local kind_url="https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/$kind_binary"
  curl -sSLo "$kind_path" $kind_url
  chmod +x "$kind_path"
}
