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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func TestDockerfileMultistage(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Dockerfile{Multistage: true})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if dockerfileMultiExp != buf.String() {
		diffs := diffutil.Diff(dockerfileMultiExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const dockerfileMultiExp = `# Binary builder image
FROM golang:1.10-alpine3.8 AS builder

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/github.com/example-inc/app-operator
COPY . /go/src/github.com/example-inc/app-operator

RUN go build -o /go/bin/app-operator github.com/example-inc/app-operator/cmd/manager

# Base image
FROM alpine:3.8

ENV OPERATOR=/usr/local/bin/app-operator \
    USER_UID=1001 \
    USER_NAME=app-operator

# install operator binary
COPY --from=builder /go/bin/app-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
`

func TestDockerfileNonMultistage(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Dockerfile{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if dockerfileNonMultiExp != buf.String() {
		diffs := diffutil.Diff(dockerfileNonMultiExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const dockerfileNonMultiExp = `FROM alpine:3.8
ENV OPERATOR=/usr/local/bin/app-operator \
    USER_UID=1001 \
    USER_NAME=app-operator

# install operator binary
COPY build/_output/bin/app-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
`
