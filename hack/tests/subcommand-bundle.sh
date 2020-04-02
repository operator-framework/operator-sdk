#!/usr/bin/env bash

source hack/lib/test_lib.sh

function cleanup_case() {
  git clean -dfxq .
}

TEST_DIR="test/test-framework"
OPERATOR_NAME="memcached-operator"
OPERATOR_VERSION_1="0.0.2"
OPERATOR_VERSION_2="0.0.3"
OPERATOR_BUNDLE_IMAGE_2="quay.io/example/${OPERATOR_NAME}:${OPERATOR_VERSION_2}"
OPERATOR_BUNDLE_ROOT_DIR="deploy/olm-catalog/${OPERATOR_NAME}"
OPERATOR_BUNDLE_DIR_1="${OPERATOR_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION_1}"
OPERATOR_BUNDLE_DIR_2="${OPERATOR_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION_2}"
OUTPUT_DIR="foo"

function create() {
  operator-sdk bundle create $1 --directory $2 --package $OPERATOR_NAME ${@:3}
}

function generate() {
  operator-sdk bundle create --generate-only --directory $1 --package $OPERATOR_NAME ${@:2}
}

function check_validate_pass() {
  if ! operator-sdk bundle validate $2 ${@:3}; then
    error_text "${1}: validate failed"
    exit 1
  fi
}

function check_validate_fail() {
  if operator-sdk bundle validate $2 ${@:3}; then
    error_text "${1}: validate passed"
    exit 1
  fi
}

pushd "$TEST_DIR"
trap_add "git clean -dfxq $TEST_DIR" EXIT
trap_add "popd" EXIT

set -e

header_text "Running 'operator-sdk bundle' subcommand tests."

TEST_NAME="create with version ${OPERATOR_VERSION_2}"
header_text "$TEST_NAME"
create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 0
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION_2} and output-dir"
header_text "$TEST_NAME"
create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2" --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 0
check_validate_pass "$TEST_NAME" "$OUTPUT_DIR"
cleanup_case

TEST_NAME="generate with version ${OPERATOR_VERSION_2}"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
check_validate_pass "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

TEST_NAME="generate with version ${OPERATOR_VERSION_2} and output-dir"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2" --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_file "$TEST_NAME" "bundle.Dockerfile" 1
check_validate_pass "$TEST_NAME" "$OUTPUT_DIR"
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION_2} with existing metadata"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
check_validate_pass "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION_2} with existing metadata and output-dir"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2" --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
check_validate_pass "$TEST_NAME" "$OUTPUT_DIR"
check_validate_pass "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

TEST_NAME="error on create with version ${OPERATOR_VERSION_2} with existing manifests version ${OPERATOR_VERSION_1}"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_1"
if create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2"; then
  error_text "$TEST_NAME: expected error"
  exit 1
fi
cleanup_case

TEST_NAME="create with version ${OPERATOR_VERSION_2} with existing manifests/metadata version ${OPERATOR_VERSION_1} and overwrite"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
create $OPERATOR_BUNDLE_IMAGE_2 "$OPERATOR_BUNDLE_DIR_2" --overwrite
check_dir "$TEST_NAME" "${OUTPUT_DIR}/manifests" 0
check_dir "$TEST_NAME" "${OUTPUT_DIR}/metadata" 0
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
check_validate_pass "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

TEST_NAME="error on validate invalid generated bundle content with version ${OPERATOR_VERSION_2}"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
# Change version to an invalid value.
sed -i 's/version: '$OPERATOR_VERSION_2'/version: a.b.c/g' "${OPERATOR_BUNDLE_ROOT_DIR}"/manifests/*.clusterserviceversion.yaml
check_validate_fail "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

TEST_NAME="error on validate invalid generated bundle format with version ${OPERATOR_VERSION_2}"
header_text "$TEST_NAME"
generate "$OPERATOR_BUNDLE_DIR_2"
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/manifests" 1
check_dir "$TEST_NAME" "${OPERATOR_BUNDLE_ROOT_DIR}/metadata" 1
# Change annotations mediatype to the incorrect type.
sed -i 's/mediatype.v1: registry+v1/mediatype.v1: plain/g' "${OPERATOR_BUNDLE_ROOT_DIR}"/metadata/annotations.yaml
check_validate_fail "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR"
cleanup_case

header_text "All 'operator-sdk bundle' subcommand tests passed."
