#!/usr/bin/env bash

if [ "$TRAVIS_COMMIT_RANGE" != "" ] && ! git diff --name-only $TRAVIS_COMMIT_RANGE | grep -qvE '(\.md)|(\.MD)|(\.png)|(\.pdf)|^(doc/)|^(MAINTAINERS)|^(LICENSE)'; then
  echo "Only doc files were updated, not running the CI."
  exit
fi

