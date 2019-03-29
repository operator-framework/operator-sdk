// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scaffold

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
)

type TestFrameworkDockerfile struct {
	input.Input
}

func (s *TestFrameworkDockerfile) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(BuildTestDir, DockerfileFile)
	}
	s.TemplateBody = testDockerfileTmpl
	return s.Input, nil
}

const testDockerfileTmpl = `# ARG before FROM must always be before the first FROM
ARG BASEIMAGE
# Test binary builder image
FROM golang:1.10-alpine3.8 AS builder

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
ENV GOFLAGS "-gcflags all=-trimpath=${GOPATH} -asmflags all=-trimpath=${GOPATH}"

WORKDIR /go/src/{{.Repo}}
COPY . /go/src/{{.Repo}}

ARG TESTDIR
RUN go test $GOFLAGS -c -o /go/bin/{{.ProjectName}}-test ${TESTDIR}/...

# Base image containing "{{.ProjectName}}-test" binary
FROM ${BASEIMAGE}
COPY --from=builder /go/bin/{{.ProjectName}}-test /usr/local/bin/{{.ProjectName}}-test

ARG NAMESPACEDMAN
COPY $NAMESPACEDMAN /namespaced.yaml
COPY build/test-framework/go-test.sh /go-test.sh
`
