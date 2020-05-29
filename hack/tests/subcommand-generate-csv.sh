#!/usr/bin/env bash

set -ee

source hack/lib/test_lib.sh

function cleanup_case() {
  git clean -dfxq . && git checkout -q .
}

TEST_DIR="test/test-framework"
OPERATOR_NAME="memcached-operator"
OPERATOR_VERSION="0.0.4"
OPERATOR_BUNDLE_ROOT_DIR="deploy/olm-catalog/${OPERATOR_NAME}"
DEFAULT_BUNDLE_DIR="${OPERATOR_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION}"
OUTPUT_DIR="foo"
OUTPUT_BUNDLE_ROOT_DIR="${OUTPUT_DIR}/olm-catalog/${OPERATOR_NAME}"
OUTPUT_BUNDLE_DIR="${OUTPUT_BUNDLE_ROOT_DIR}/${OPERATOR_VERSION}"

function check_csv_file_legacy() {
  check_file "$1" "${2}/${OPERATOR_NAME}.v${OPERATOR_VERSION}.clusterserviceversion.yaml" $3
}

function check_csv_file() {
  check_file "$1" "${2}/${OPERATOR_NAME}.clusterserviceversion.yaml" $3
}

function check_package_file() {
  check_file "$1" "${2}/${OPERATOR_NAME}.package.yaml" $3
}

function check_crd_files() {
  local memcacheds_crd_file="${2}/cache.example.com_memcacheds_crd.yaml"
  local memcachedrs_crd_file="${2}/cache.example.com_memcachedrs_crd.yaml"
  for file in $memcacheds_crd_file $memcachedrs_crd_file; do check_file "$1" "$file" $3; done
}

function generate_csv() {
  echo_run operator-sdk generate csv --operator-name $OPERATOR_NAME --interactive=false $@
}

function generate_bundle() {
  echo_run operator-sdk generate bundle --operator-name $OPERATOR_NAME --interactive=false $@
}

function generate_packagemanifests() {
  echo_run operator-sdk generate packagemanifests --operator-name $OPERATOR_NAME --interactive=false $@
}

pushd "$TEST_DIR" > /dev/null
trap_add "git clean -dfxq $TEST_DIR" EXIT
trap_add "popd > /dev/null" EXIT

header_text "Running 'operator-sdk generate csv' subcommand tests in $TEST_DIR."

TEST_NAME="generate with version $OPERATOR_VERSION"
header_text "$TEST_NAME"
generate_csv --make-manifests=false --csv-version $OPERATOR_VERSION
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 1
check_csv_file_legacy "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 1
check_crd_files "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and output-dir"
header_text "$TEST_NAME"
generate_csv --make-manifests=false --csv-version $OPERATOR_VERSION --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 1
check_csv_file_legacy "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 1
check_crd_files "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 0
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and output-dir, update-crds"
header_text "$TEST_NAME"
generate_csv --make-manifests=false --csv-version $OPERATOR_VERSION --output-dir "$OUTPUT_DIR" --update-crds
check_dir "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 1
check_csv_file_legacy "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 1
check_crd_files "$TEST_NAME" "$OUTPUT_BUNDLE_DIR" 1
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and make-manifests"
header_text "$TEST_NAME"
generate_csv --csv-version $OPERATOR_VERSION
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_csv_file "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_crd_files "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and output-dir, make-manifests"
header_text "$TEST_NAME"
generate_csv --csv-version $OPERATOR_VERSION --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_dir "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
check_csv_file "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
check_crd_files "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
cleanup_case

header_text "All 'operator-sdk generate csv' subcommand tests passed."

header_text "Running 'operator-sdk generate bundle' subcommand tests in $TEST_DIR."

TEST_NAME="generate with version $OPERATOR_VERSION"
header_text "$TEST_NAME"
generate_bundle --version $OPERATOR_VERSION
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 1
check_csv_file "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_crd_files "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

TEST_NAME="generate manifests only with version $OPERATOR_VERSION"
header_text "$TEST_NAME"
generate_bundle --version $OPERATOR_VERSION --manifests
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 0
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/metadata" 0
check_csv_file "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_crd_files "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 0
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and output-dir"
header_text "$TEST_NAME"
generate_bundle --version $OPERATOR_VERSION --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR/manifests" 0
check_dir "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
check_dir "$TEST_NAME" "$OUTPUT_DIR/metadata" 1
check_csv_file "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
check_crd_files "$TEST_NAME" "$OUTPUT_DIR/manifests" 1
check_file "$TEST_NAME" "bundle.Dockerfile" 1
cleanup_case

header_text "All 'operator-sdk generate bundle' subcommand tests passed."

header_text "Running 'operator-sdk generate packagemanifests' subcommand tests in $TEST_DIR."

TEST_NAME="generate with version $OPERATOR_VERSION"
header_text "$TEST_NAME"
generate_packagemanifests --version $OPERATOR_VERSION
check_dir "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 1
check_package_file "$TEST_NAME" "$OPERATOR_BUNDLE_ROOT_DIR" 1
check_csv_file "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 1
check_crd_files "$TEST_NAME" "$DEFAULT_BUNDLE_DIR" 1
cleanup_case

TEST_NAME="generate with version $OPERATOR_VERSION and output-dir"
header_text "$TEST_NAME"
generate_packagemanifests --version $OPERATOR_VERSION --output-dir "$OUTPUT_DIR"
check_dir "$TEST_NAME" "${OUTPUT_DIR}/${OPERATOR_VERSION}" 1
check_package_file "$TEST_NAME" "$OUTPUT_DIR" 1
check_csv_file "$TEST_NAME" "${OUTPUT_DIR}/${OPERATOR_VERSION}" 1
check_crd_files "$TEST_NAME" "${OUTPUT_DIR}/${OPERATOR_VERSION}" 1
cleanup_case

header_text "All 'operator-sdk generate packagemanifests' subcommand tests passed."
