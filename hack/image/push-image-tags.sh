#!/usr/bin/env bash
source hack/lib/image_lib.sh

#
# push_image_tags <source_image> <push_image>
#
# push_image_tags tags the source docker image with zero or more
# image tags based on TravisCI environment variables and the
# presence of git tags in the repository of the current working
# directory. If a second argument is present, it will be used as
# the base image name in pushed image tags.
#
function push_image_tags() {

  source_image=$1; shift || fatal "${FUNCNAME} usage error"
  push_image=$1; shift || push_image=$source_image

  print_image_info $source_image
  print_git_tags

  docker_login $push_image

  check_can_push || return 0

  images=$(get_image_tags $push_image)

  for image in $images; do
    docker tag "$source_image" "$image"
    docker push "$image"
  done
}

#
# print_image_info <image_name>
#
# print_image_info prints helpful information about a docker
# image.
#
function print_image_info() {
  image_name=$1; shift || fatal "${FUNCNAME} usage error"
  image_id=$(docker inspect "$image_name" -f "{{.Id}}")
  image_created=$(docker inspect "$image_name" -f "{{.Created}}")

  if [[ -n "$image_id" ]]; then
    echo "Docker image info:"
    echo "    Name:      $image_name"
    echo "    ID:        $image_id"
    echo "    Created:   $image_created"
    echo ""
  else
    echo "Could not find docker image \"$image_name\""
    return 1
  fi
}

#
# print_git_tags
#
# print_git_tags prints all tags present in the git repository.
#
function print_git_tags() {
  git_tags=$(git tag -l | sed 's|^|    |')
  if [[ -n "$git_tags" ]]; then
    echo "Found git tags:"
    echo "$git_tags"
    echo ""
  fi
}


#
# latest_git_version
#
# latest_git_version returns the highest semantic version
# number found in the repository, with the form "vX.Y.Z".
# Version numbers not matching the semver release format
# are ignored.
#
function latest_git_version() {
  git tag -l | egrep "${semver_regex}" | sort -V | tail -1
}

push_image_tags "$@"
