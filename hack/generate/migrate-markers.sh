#!/usr/bin/env bash

# Migrate old CSV annotations to new markers.
# Example:
# $ ./migrate-markers.sh *.go

OLD_MARKER_BASE="operator-sdk:gen-csv:customresourcedefinitions"
NEW_MARKER_BASE="operator-sdk:csv:customresourcedefinitions"

SPEC_TYPE="spec"
STATUS_TYPE="status"

function migrate_displayName() {
  local old_marker_pattern="${OLD_MARKER_BASE}"'\.displayName="([^"]+)"'
  local new_marker_pattern="${NEW_MARKER_BASE}"':displayName="\1"'

  sed -i -E 's/'$old_marker_pattern'/'$new_marker_pattern'/g' "$1"
}

function migrate_resources() {
  local old_marker_pattern_1="${OLD_MARKER_BASE}"'\.resources="([^\,]+)\,([^\,]+)\,\\"([^\\]+)\\""'
  local old_marker_pattern_2="${OLD_MARKER_BASE}"'\.resources="([^\,]+)\,([^\,]+)\,"'
  local new_marker_pattern_1="${NEW_MARKER_BASE}"':resources={\1,\2,"\3"}'
  local new_marker_pattern_2="${NEW_MARKER_BASE}"':resources={\1,\2,}'

  sed -i -E 's/'$old_marker_pattern_1'/'$new_marker_pattern_1'/g' "$1"
  sed -i -E 's/'$old_marker_pattern_2'/'$new_marker_pattern_2'/g' "$1"
}

function migrate_typeDescriptors() {
  local type=$2
  local old_marker_true_pattern="${OLD_MARKER_BASE}\.${type}Descriptors=true"
  local old_marker_false_pattern="${OLD_MARKER_BASE}\.${type}Descriptors=false"
  local new_marker_pattern="${NEW_MARKER_BASE}:type=${type}"

  sed -i -E 's/'$old_marker_true_pattern'/'$new_marker_pattern'/g' "$1"
  sed -i -E 's/'$old_marker_false_pattern'//g' "$1"
}

function migrate_typeDescriptors_displayName() {
  local type=$2
  local old_marker_pattern="${OLD_MARKER_BASE}\.${type}Descriptors"'\.displayName="([^"]+)"'
  local new_marker_pattern="${NEW_MARKER_BASE}:type=${type}"',displayName="\1"'

  sed -i -E 's/'$old_marker_pattern'/'$new_marker_pattern'/g' "$1"
}

function migrate_typeDescriptors_xDescriptors() {
  local type=$2
  local old_marker_pattern="${OLD_MARKER_BASE}\.${type}Descriptors"'\.x-descriptors="([^"]+)"'
  local new_marker_pattern="${NEW_MARKER_BASE}:type=${type}"',xDescriptors="\1"'

  sed -i -E 's/'$old_marker_pattern'/'$new_marker_pattern'/g' "$1"
}

set -eu
shopt -s extglob nullglob
FILES=$*
shopt -u extglob nullglob

for file in $FILES; do
  if ! [[ "$file" =~ .*\.go ]]; then
    continue
  fi
  # Globals
  migrate_displayName "$file"
  migrate_resources "$file"
  # Spec descriptors
  migrate_typeDescriptors "$file" $SPEC_TYPE
  migrate_typeDescriptors_displayName "$file" $SPEC_TYPE
  migrate_typeDescriptors_xDescriptors "$file" $SPEC_TYPE
  # Status descriptors
  migrate_typeDescriptors "$file" $STATUS_TYPE
  migrate_typeDescriptors_displayName "$file" $STATUS_TYPE
  migrate_typeDescriptors_xDescriptors "$file" $STATUS_TYPE
done
