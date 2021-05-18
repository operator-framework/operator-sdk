#!/usr/bin/env bash

set -eu
set -o pipefail

# Major version to select (default 1).
MAJ=${1:-1}
# Number of minor versions to select (default 2).
NUM_MINORS=${2:-2}

# Get unique "v${major}.${minor}" tags, then add the greatest patch version for each
# to a list.
declare -a LATEST_GIT_TAGS
for tag in $(git tag --sort=-v:refname -l "v${MAJ}.*" | grep -Eo "v${MAJ}\.[^\.]+" | uniq | head -n $NUM_MINORS); do
  LATEST_GIT_TAGS+=( $(git tag --sort=-v:refname -l "$tag*" | head -n 1) )
done
# Print tags in comma-separated form.
echo ${LATEST_GIT_TAGS[@]} | sed -E 's/[ ]+/,/g'
