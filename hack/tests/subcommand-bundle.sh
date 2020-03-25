#!/usr/bin/env bash

source hack/lib/test_lib.sh

function check_dir() {
  if [[ $3 == 1 ]]; then
    if [[ -d "$2" ]]; then
      echo "${1}: directory ${2} should not exist"
      exit 1
    fi
  else
    if [[ ! -d "$2" ]]; then
      echo "${1}: directory ${2} should exist"
      exit 1
    fi
  fi
}

function check_file() {
  if [[ $3 == 1 ]]; then
    if [[ -f "$2" ]]; then
      echo "${1}: file ${2} should not exist"
      exit 1
    fi
  else
    if [[ ! -f "$2" ]]; then
      echo "${1}: file ${2} should exist"
      exit 1
    fi
  fi
}

function cleanup() {
  git clean -dfx test/test-framework
}

OPERATOR_NAME="memcached-operator"
OPERATOR_VERSION="0.0.3"
OPERATOR_BUNDLE_IMAGE="quay.io/example/${OPERATOR_NAME}:${OPERATOR_VERSION}"
OPERATOR_BUNDLE_ROOT_DIR="deploy/olm-catalog/${OPERATOR_NAME}"
OPERATOR_BUNDLE_DIR="${OPERATOR_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION}"
CREATE_CMD="operator-sdk bundle create $OPERATOR_BUNDLE_IMAGE"
GENERATE_CMD="operator-sdk bundle create --generate-only"

pushd test/test-framework
trap_add "cleanup" EXIT
trap_add "popd" EXIT

set -ex

cp $(find deploy/crds/ -name *memcached*_crd.yaml) "${OPERATOR_BUNDLE_DIR}"

TEST_NAME="create with version ${OPERATOR_VERSION}"
$CREATE_CMD --version $OPERATOR_VERSION --directory "$OPERATOR_BUNDLE_ROOT_DIR" --package $OPERATOR_NAME
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1

TEST_NAME="create with latest"
$CREATE_CMD --latest --directory "$OPERATOR_BUNDLE_ROOT_DIR" --package $OPERATOR_NAME
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1

TEST_NAME="generate with version ${OPERATOR_VERSION}"
$GENERATE_CMD --version $OPERATOR_VERSION --directory "$OPERATOR_BUNDLE_ROOT_DIR" --package $OPERATOR_NAME
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 0
cleanup

TEST_NAME="create with version ${OPERATOR_VERSION} with manifests dir"
$GENERATE_CMD --version $OPERATOR_VERSION --directory "$OPERATOR_BUNDLE_ROOT_DIR" --package $OPERATOR_NAME
rm -rf "$OPERATOR_BUNDLE_ROOT_DIR/metadata" "bundle.Dockerfile"
$CREATE_CMD --directory "$OPERATOR_BUNDLE_ROOT_DIR" --package $OPERATOR_NAME
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup

# TODO(estroz): add validate steps after each 'create' test to validate dirs
# once the following is merged:
# https://github.com/operator-framework/operator-sdk/pull/2737
