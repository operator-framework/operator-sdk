#!/usr/bin/env bash

# _buildx runs "docker buildx build" in CI-output mode.
function _buildx() {
  echo -e "\n$ docker buildx build --progress plain $@"
  docker buildx build --progress plain $@
}

# _pull runs "docker pull".
function _pull() {
  echo -e "\n$ docker pull $@"
  docker pull $@
}

# cmp_times returns false if $2 occurred within some timespan relative to $1.
# The TIMESPAN variable
function cmp_times() {
  local base_seconds=$(date -d "$1" +%s)
  local img_time_seconds=$(date -d "$2" +%s)
  if (( $base_seconds - $TIMESPAN < $img_time_seconds )) || (( $FORCE )); then
    return 1
  fi
  return 0
}

# is_dockerfile_fresh returns false if at least one image in a "FROM" directive
# in the Dockerfile at $1 has been freshly built within TIMESPAN relative to now,
# or FORCE=1.
function is_dockerfile_fresh() {
  local dockerfile=$1
  # Strip flag from FROM to get image, which always precedes this flag if set.
  local docker_images=$(grep -oP "FROM (--platform=[^ ]+ )?\K([^ ]+)" $dockerfile)

  for img in $docker_images; do
    _pull $img
    local img_create_time=$(docker inspect --format '{{.Created}}' $img)
    if [[ "$img_create_time" == "0001-01-01T00:00:00Z" ]]; then 
      echo "image creation time could be found for $img"
      exit 1
    fi
    if ! cmp_times "$(date)" "$img_create_time"; then
      return 1
    fi
  done
}

# Build an image at path ./images/ansible-operator/base.Dockerfile checked out at git tag $1
# for all platforms in $2. Semantics are otherwise the same as build_generic.
function build_ansible_base() {
  local tag=$1
  local platforms=$2
  local dockerfile=./images/ansible-operator/base.Dockerfile

  git checkout refs/tags/$tag
  local ansible_base_image_tag=$(grep -oP 'FROM \K(quay\.io/estroz/ansible-operator-base:.+)' ./images/ansible-operator/Dockerfile)
  # Attempt to get the git ref that built this image from the git_commit image label,
  # falling back to parsing it from the image tag, which typically contains a git ref
  # as the last hyphen-delimit element.
  local ansible_base_git_ref=$(docker inspect --format '{{ index .Config.Labels "git_commit" }}' $ansible_base_image_tag)
  if [[ $ansible_base_git_ref == "devel" || $ansible_base_git_ref == "" ]]; then
    ansible_base_git_ref=$(echo $ansible_base_image_tag | sed -E 's|.+:.+-(.+)|\1|')
  fi
  git checkout $ansible_base_git_ref
  if ! is_dockerfile_fresh "$dockerfile"; then
    _buildx --tag $ansible_base_image_tag --platform "$platforms" --file "$dockerfile" $IMAGE_DO --build-arg GIT_COMMIT=$ansible_base_git_ref ./images/ansible-operator
  fi
}

# Build an image at path ./images/$2/Dockerfile checked out at git tag $1
# for all platforms in $3. Tag is assumed to be "v"+semver; the image is tagged
# with the full semver string and with "v${major}.${minor}".
# The build will only run if the Dockerfile is not fresh.
function build_generic() {
  local tag=$1
  local id=$2
  local platforms=$3
  local tag_maj_min="quay.io/estroz/${id}:$(echo $tag | grep -Eo "v[1-9]+\.[0-9]+")"
  local tag_full="quay.io/estroz/${id}:${tag}"
  local dockerfile=./images/${id}/Dockerfile

  git checkout refs/tags/$tag
  if ! is_dockerfile_fresh "$dockerfile"; then
    _buildx --tag "$tag_maj_min" --tag "$tag_full"  --platform "$platforms" --file "$dockerfile" $IMAGE_DO .
  fi
}

set -eu
set -o pipefail

# Comman-separated list of build platforms, ex. linux/s390x.
# See 'docker buildx build --help' for --platform flag info.
DEFAULT_PLATFORMS="linux/amd64,linux/arm64,linux/ppc64le,linux/s390x"
# TODO(estroz): support scorecard-test-kuttl rebuilds.
# DEFAULT_SCORECARD_KUTTL_PLATFORMS="linux/amd64,linux/arm64,linux/ppc64le"
PLATFORMS=
# Time window to compare image creation times against, relative to now.
DEFAULT_TIMESPAN=86400 # 24 hours in seconds
# What to do with the image, either load (default) or push.
IMAGE_DO=--load
# Space-separated list of git tags.
# The --tags arg can be comma-separated.
TAGS=
# ID of the image, ex. operator-sdk, ansible-operator.
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
  TAGS=$(echo $2 | sed -E 's/,/ /g')
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

set -x

case $IMAGE_ID in
ansible-operator)
  # ansible-operator has a base image that must be rebuilt in advance if necessary.
  # This script will detect that the base is fresh when inspecting ansible-operator's
  # Dockerfile and build it.
  for tag in $TAGS; do
    build_ansible_base $tag "$PLATFORMS"
  done
;;
esac

# Build the image defined by IMAGE_ID for each tag for a set of platforms.
for tag in $TAGS; do
  build_generic $tag $IMAGE_ID "$PLATFORMS"
done

popd
