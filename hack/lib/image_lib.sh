#!/usr/bin/env bash

source hack/lib/common.sh

semver_regex="^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"

# docker_login <image_name>
#
# docker_login performs a docker login for the server of the provided
# image if the DOCKER_USERNAME and DOCKER_PASSWORD environment variables
# are set.
#
function docker_login() {
  image_name=$1; shift || fatal "${FUNCNAME} usage error"

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

#
# load_image_if_kind <image tag>
#
# load_image_if_kind loads an image into all nodes in a kind cluster.
#
function load_image_if_kind() {
  if [[ "$(kubectl config current-context)" == "kind-kind" ]]; then
    if which kind 2>/dev/null; then
      kind load docker-image "$1"
    fi
  fi
}
