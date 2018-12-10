#!/usr/bin/env bash

source hack/lib/common.sh

semver_regex="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"

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
  docker_login
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
# docker_login <image_name>
#
# docker_login performs a docker login for the server of the provided
# image if the DOCKER_USERNAME and DOCKER_PASSWORD environment variables
# are set.
#
function docker_login() {
  if [[ -n "$DOCKER_USERNAME" && -n "$DOCKER_PASSWORD" ]]; then
    server=$(docker_server_for_image $image_name)
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin "$server"
  fi
}

#
# check_can_push
#
# check_can_push performs various checks to determine whether images
# built from this commit should be pushed. It prints a message and 
# returns a failure code if any check doesn't pass.
#
function check_can_push() {
  if [[ "$TRAVIS" != "true" ]]; then
    echo "Detected execution in a non-TravisCI environment. Skipping image push."
    return 1
  elif [[ "$TRAVIS_EVENT_TYPE" == "pull_request" ]]; then
    echo "Detected pull request commit. Skipping image push"
    return 1
  elif [[ ! -f "$HOME/.docker/config.json" ]]; then
    echo "Docker login credentials required to push. Skipping image push."
    return 1
  fi
}

#
# get_image_tags <image_name>
#
# get_image_tags returns a list of tags that are eligible to be pushed.
# If an image name is passed as an argument, the full <name>:<tag> will
# be returned for each eligible tag. The criteria is:
#   1. Is TRAVIS_BRANCH set?                 => <image_name>:$TRAVIS_BRANCH
#   2. Is TRAVIS_TAG highest semver release? => <image_name>:latest
#
function get_image_tags() {
  image_name=$1
  [[ -n "$image_name" ]] && image_name="${image_name}:"

  # Tag `:$TRAVIS_BRANCH` if it is set.
  # Note that if the build is for a tag, $TRAVIS_BRANCH is set
  # to the tag, so this works in both cases
  if [[ -n "$TRAVIS_BRANCH" ]]; then
    echo "${image_name}${TRAVIS_BRANCH}"
  fi

  # Tag `:latest` if $TRAVIS_TAG is the highest semver tag found in
  # the repository.
  if is_latest_tag "$TRAVIS_TAG"; then
    echo "${image_name}latest"
  fi
}

#
# docker_server_for_image <image_name>
#
# docker_server_for_image returns the server component of the image
# name. If the image name does not contain a server component, an
# empty string is returned.
#
function docker_server_for_image() {
  image_name=$1; shift || fatal "${FUNCNAME} usage error"
  IFS='/' read -r -a segments <<< "$image_name"
  if [[ "${#segments[@]}" -gt "2" ]]; then
    echo "${segments[0]}"
  else
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

#
# is_latest_tag <candidate>
#
# is_latest_tag returns whether the candidate tag matches
# the latest tag from the git repository, based on semver.
# To be the latest tag, the candidate must match the semver
# release format.
#
function is_latest_tag() {
  candidate=$1; shift || fatal "${FUNCNAME} usage error"
  if ! [[ "$candidate" =~ $semver_regex ]]; then
    return 1
  fi

  latest="$(latest_git_version)"
  [[ -z "$latest" || "$candidate" == "$latest" ]]
}

push_image_tags "$@"
