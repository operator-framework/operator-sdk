#!/usr/bin/env bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

source ${DIR}/lib.sh

set -eu
set -o pipefail

# Comma-separated list of build platforms, ex. linux/s390x.
# See 'docker buildx build --help' for --platform flag info.
DEFAULT_PLATFORMS="linux/amd64,linux/arm64,linux/ppc64le,linux/s390x"
# TODO(estroz): support scorecard-test-kuttl rebuilds.
# DEFAULT_SCORECARD_KUTTL_PLATFORMS="linux/amd64,linux/arm64,linux/ppc64le"
PLATFORMS=
# Time window to compare image creation times against, relative to now.
# --timespan should be set to this duration in seconds.
DEFAULT_TIMESPAN=86400 # 24 hours in seconds
TIMESPAN=
# What to do with the image, either load (default) or push.
IMAGE_DO=--load
# Space-separated list of git tags.
# The --tags arg can be comma-separated.
TAGS=
# ID of the image, ex. operator-sdk.
IMAGE_ID=
# Update all images.
FORCE=0
while [[ $# -gt 0 ]]; do
  case $1 in
  --push)
  IMAGE_DO=$1
  ;;
  --force)
  FORCE=1
  ;;
  --tags)
    TAGS=($(echo $2 | sed -E 's/,/ /g'))
  shift
  ;;
  --image-id)
  IMAGE_ID=$2
  shift
  ;;
  --platforms)
  PLATFORMS=$2
  shift
  ;;
  --timespan)
  TIMESPAN=$2
  shift
  ;;
  *) echo "Invalid flag $1"; exit 1 ;;
  esac
  shift
done

: ${IMAGE_ID:?--image-id is required}
: ${TAGS:?--tags is required}

# Set defaults.
case $IMAGE_ID in
scorecard-test-kuttl)
  # TODO(estroz): support scorecard-test-kuttl rebuilds.
  # PLATFORMS=${PLATFORMS:-$DEFAULT_SCORECARD_KUTTL_PLATFORMS}
  echo "$IMAGE_ID is not supported"
  exit 1
;;
*)
  PLATFORMS=${PLATFORMS:-$DEFAULT_PLATFORMS}
;;
esac
TIMESPAN=${TIMESPAN:-$DEFAULT_TIMESPAN}

# Clone the operator-sdk repo into a temp dir with cleanup.
tmp=$(mktemp -d --tmpdir freshen-images-tmp.XXXXXX)
git clone https://github.com/operator-framework/operator-sdk.git $tmp
trap "rm -rf $tmp" EXIT
pushd $tmp


# Build the image defined by IMAGE_ID for each tag for a set of platforms.
for i in ${!TAGS[*]}; do
  if (($i=0)); then
    build_generic ${TAGS[$i]} $IMAGE_ID "$PLATFORMS" true
  else
    build_generic ${TAGS[$i]} $IMAGE_ID "$PLATFORMS" false
  fi
done

popd
