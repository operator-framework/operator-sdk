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

const dockerfileTmpl = `
#############################################
# Temp image to build the prerequisites 
#############################################
FROM registry.access.redhat.com/ubi8/ubi:latest AS builder

RUN groupadd -g 1001 {{.ProjectName}} && \
    useradd -m -s /sbin/nologin -u 1001 -g {{.ProjectName}} {{.ProjectName}}

#############################################
# Container Image
#############################################
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# Set EnvVars
ENV OPERATOR=/usr/local/bin/{{.ProjectName}} \
    USER_UID=1001 \
    USER_NAME={{.ProjectName}}

# Copy group and user from builder image
COPY --from=builder /etc/passwd /tmp
COPY --from=builder /etc/group /tmp

# Create user and add permissions
RUN cat /tmp/passwd | grep {{.ProjectName}} >> /etc/passwd \
    && cat /tmp/group | grep {{.ProjectName}} >> /etc/group \
    && mkdir -p /home/{{.ProjectName}} \
    && chown -R {{.ProjectName}}:{{.ProjectName}} /home/{{.ProjectName}} \
    && chmod ug+rwx /home/{{.ProjectName}} \
	&& rm -rf /tmp/passwd \
    && rm -rf /tmp/group

USER ${USER_UID}

# Install operator binary
COPY build/_output/bin/{{.ProjectName}} ${OPERATOR}
COPY build/bin /usr/local/bin

ENTRYPOINT ["/usr/local/bin/entrypoint"]`