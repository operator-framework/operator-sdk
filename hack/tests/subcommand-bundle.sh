#!/usr/bin/env bash

source hack/lib/test_lib.sh

function check_dir() {
  if [[ $3 == 0 ]]; then
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
  if [[ $3 == 0 ]]; then
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

function cleanup_case() {
  git clean -dfx .
}

function cp_crds() {
  cp $(find deploy/crds/ -name *memcached*_crd.yaml) "$OPERATOR_BUNDLE_DIR"
}

TEST_DIR="test/test-framework"
OPERATOR_NAME="memcached-operator"
OPERATOR_VERSION="0.0.3"
OPERATOR_BUNDLE_IMAGE="quay.io/example/${OPERATOR_NAME}:${OPERATOR_VERSION}"
OPERATOR_BUNDLE_ROOT_DIR="deploy/olm-catalog/${OPERATOR_NAME}"
OPERATOR_BUNDLE_DIR="${OPERATOR_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION}"
OUTPUT_DIR="foo"
CREATE_CMD="operator-sdk bundle create $OPERATOR_BUNDLE_IMAGE --directory "$OPERATOR_BUNDLE_DIR" --package $OPERATOR_NAME"
GENERATE_CMD="operator-sdk bundle create --generate-only --directory "$OPERATOR_BUNDLE_DIR" --package $OPERATOR_NAME"

pushd "$TEST_DIR"
# trap_add "git clean -dfx $TEST_DIR" EXIT
trap_add "popd" EXIT

set -ex

TEST_NAME="create with version ${OPERATOR_VERSION}"
cp_crds
$CREATE_CMD
check_dir "$TEST_NAME" "${OUTPUT_DIR}" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_DIR/metadata" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 0
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION} and output-dir"
cp_crds
$CREATE_CMD --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 0
cleanup_case

TEST_NAME="generate with version ${OPERATOR_VERSION}"
cp_crds
$GENERATE_CMD
check_dir "$TEST_NAME" "${OUTPUT_DIR}" 0
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

TEST_NAME="generate with version ${OPERATOR_VERSION} and output-dir"
cp_crds
$GENERATE_CMD --output-dir "$OUTPUT_DIR"
ls -alR "$OPERATOR_BUNDLE_DIR"
ls -alR "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION} with existing metadata"
cp_crds
$GENERATE_CMD
$CREATE_CMD
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION} with existing metadata and output-dir"
cp_crds
$GENERATE_CMD
$CREATE_CMD --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION} from output-dir and output-dir"
cp_crds
$GENERATE_CMD --output-dir "$OUTPUT_DIR"
operator-sdk bundle create $OPERATOR_BUNDLE_IMAGE --package $OPERATOR_NAME --directory "${OUTPUT_DIR}/manifests"
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 0
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

# TODO(estroz): add validate steps after each 'create' test to validate dirs
# once the following is merged:
# https://github.com/operator-framework/operator-sdk/pull/2737
