package generator

const buildTmpl = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/tmp/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="{{.ProjectName}}"
REPO_PATH="{{.RepoPath}}"
BUILD_PATH="${REPO_PATH}/cmd/${PROJECT_NAME}"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`

const dockerBuildTmpl = `#!/usr/bin/env bash

if ! which docker > /dev/null; then
	echo "docker needs to be installed"
	exit 1
fi

: ${IMAGE:?"Need to set IMAGE, e.g. gcr.io/<repo>/<your>-operator"}

echo "building container ${IMAGE}..."
docker build -t "${IMAGE}" -f tmp/build/Dockerfile .
`

const dockerFileTmpl = `FROM alpine:3.6

ADD tmp/_output/bin/{{.ProjectName}} /usr/local/bin/{{.ProjectName}}

RUN adduser -D {{.ProjectName}}
USER {{.ProjectName}}
`
