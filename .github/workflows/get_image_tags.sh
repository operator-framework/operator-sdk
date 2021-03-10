#!/usr/bin/env bash

IMG="$1"
TAG_PREFIX="$2"

: ${IMG:?"\$1 must be set to an image tag"}
: ${TAG_PREFIX:?"\$2 must be set to some tag prefix to pass to refs/tags/{prefix}*"}
: ${GITHUB_REF:?"GITHUB_REF must be set to a git 'refs/' path in the environment (typically set by the Actions runner)"}

if [[ $GITHUB_REF == refs/tags/${TAG_PREFIX}* ]]; then
  # Release tags.
  TAG="${GITHUB_REF#refs/tags/${TAG_PREFIX}}"
  # Prepend "v" if removed by the above variable operation, since $TAG should always be semver.
  [[ $TAG == v* ]] || TAG="v${TAG}"
  MAJOR_MINOR="${TAG%.*}"
  echo "${IMG}:${TAG},${IMG}:${MAJOR_MINOR}"

elif [[ $GITHUB_REF == refs/tags/* ]]; then
  # Any other tag, which will not be pushed.
  TAG="$(echo "${GITHUB_REF#refs/tags/}" | sed -r 's|/+|-|g')-local"
  echo "${IMG}:${TAG}"

elif [[ $GITHUB_REF == refs/heads/* ]]; then
  # Branch build.
  TAG="$(echo "${GITHUB_REF#refs/heads/}" | sed -r 's|/+|-|g')"
  echo "${IMG}:${TAG}"

elif [[ $GITHUB_REF == refs/pull/* ]]; then
  # PR build.
  TAG="pr-$(echo "${GITHUB_REF}" | sed -E 's|refs/pull/([^/]+)/?.*|\1|')"
  echo "${IMG}:${TAG}"
fi
