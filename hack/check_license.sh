#!/bin/bash
echo "Checking for license header..."
allfiles=$(go list ./commands/... ./pkg/... ./test/... | grep -v generated | xargs -I {} find "${GOPATH}/src/{}" -name '*.go' | grep -v generated)
licRes=""
for file in $allfiles; do
  if ! head -n3 "${file}" | grep -Eq "(Copyright|generated|GENERATED)" ; then
    licRes="${licRes}\n"$(echo -e "  ${file}")
  fi
done
if [ -n "${licRes}" ]; then
  echo -e "license header checking failed:\n${licRes}"
  exit 255
fi
