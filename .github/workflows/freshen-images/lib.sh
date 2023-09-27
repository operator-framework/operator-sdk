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

# cmp_times returns false if time $2 occurred within some timespan defined by TIMESPAN
# relative to time $1.
function cmp_times() {
  local base_seconds=$(date -d "$1" +%s)
  local img_time_seconds=$(date -d "$2" +%s)
  if (( $base_seconds - $TIMESPAN < $img_time_seconds )) || (( $FORCE )); then
    # return false
    return 1
  fi
  # return true
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
      # return false
      echo "is_dockerfile_fresh returning 1 (false) for [$img]"
      return 1
    fi
  done
}


# Build an image at path ./images/$2/Dockerfile checked out at git tag $1
# for all platforms in $3. Tag is assumed to be "v"+semver; the image is tagged
# with the full semver string and with "v${major}.${minor}".
# The build will only run if the Dockerfile is not fresh.
function build_generic() {
  local tag=$1
  local id=$2
  local platforms=$3
  local buildlatest=$4
  local tag_maj_min="quay.io/operator-framework/${id}:$(echo $tag | grep -Eo "v[1-9]+\.[0-9]+")"
  local tag_full="quay.io/operator-framework/${id}:${tag}"
  local tag_latest="quay.io/operator-framework/${id}:latest"
  local dockerfile=./images/${id}/Dockerfile

  git checkout refs/tags/$tag
  if is_dockerfile_fresh "$dockerfile"; then
    echo "Skipping build of $dockerfile, it is FRESH!"
  else
    # dockerfile is not fresh, rebuildng image
    if $buildlatest; then
      echo "Rebuilding image [$tag_maj_min] and latest for [$platforms]"
      _buildx --builder=container --tag "$tag_maj_min" --tag "$tag_full"  --tag "$tag_latest" --platform "$platforms" --file "$dockerfile" $IMAGE_DO .
    else
      echo "Rebuilding image [$tag_maj_min] for [$platforms]"
      _buildx --builder=container --tag "$tag_maj_min" --tag "$tag_full"  --platform "$platforms" --file "$dockerfile" $IMAGE_DO .
    fi
  fi
}
