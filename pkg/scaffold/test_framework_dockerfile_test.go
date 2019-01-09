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

func TestTestFrameworkDockerfileMultistage(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &TestFrameworkDockerfile{})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if testDockerfileExp != buf.String() {
		diffs := diffutil.Diff(testDockerfileExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const testDockerfileExp = `# ARG before FROM must always be before the first FROM
ARG BASEIMAGE
# Test binary builder image
FROM golang:1.10-alpine3.8 AS builder

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/github.com/example-inc/app-operator
COPY . /go/src/github.com/example-inc/app-operator

ARG TESTDIR
RUN go test -c -o /go/bin/app-operator-test ${TESTDIR}/...

# Base image containing "app-operator-test" binary
FROM ${BASEIMAGE}
COPY --from=builder /go/bin/app-operator-test /usr/local/bin/app-operator-test

ARG NAMESPACEDMAN
COPY $NAMESPACEDMAN /namespaced.yaml
COPY build/test-framework/go-test.sh /go-test.sh
`
