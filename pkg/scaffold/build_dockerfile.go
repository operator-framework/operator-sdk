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

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const DockerfileFile = "Dockerfile"

type Dockerfile struct {
	input.Input
}

func (s *Dockerfile) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(BuildDir, DockerfileFile)
	}
	s.TemplateBody = dockerfileTmpl
	return s.Input, nil
}

const dockerfileTmpl = `# Binary builder image
FROM golang:1.10.3-alpine3.8 AS builder

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/{{ .Repo }}
COPY . /go/src/{{ .Repo }}

RUN go build -o /go/bin/{{ .ProjectName }} {{ .Repo }}/cmd/manager

# Base image
FROM alpine:3.8

RUN apk upgrade --update --no-cache
USER nobody

COPY --from=builder /go/bin/{{ .ProjectName }} /usr/local/bin/{{ .ProjectName }}
`
