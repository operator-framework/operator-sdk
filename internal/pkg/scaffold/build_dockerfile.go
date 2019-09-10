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
MAINTAINER Red Hat Operator Framework <operator-framework@googlegroups.com>

# Define Labels
LABEL name="operator-framework/operator-sdk-operator-image" \
      maintainer="operator-framework@googlegroups.com" \
      vendor="Operator Framework" 

# Define EnvVars
ENV OPERATOR=/usr/local/bin/{{.ProjectName}} \
    USER_UID=1001 \
    USER_NAME={{.ProjectName}} \
    USER_HOME=/home/{{.ProjectName}}  

# Copy group and user created in the builder image
COPY --from=builder /etc/passwd /tmp
COPY --from=builder /etc/group /tmp

# Create the user and group in this image and add permissions 
RUN cat /tmp/passwd | grep ${USER_NAME} >> /etc/passwd \
    && cat /tmp/group | grep ${USER_NAME} >> /etc/group \
    && mkdir -p ${USER_HOME} \
    && chown -R ${USER_NAME}:${USER_NAME} ${USER_HOME} \
    && chmod ug+rwx ${USER_HOME} \
    && rm -rf /tmp/passwd \
    && rm -rf /tmp/group

# Use the user created to run the container as rootless
USER ${USER_UID}

# Installs the operator binary
COPY build/_output/bin/${USER_NAME} ${OPERATOR}
COPY build/bin /usr/local/bin

# This allows the Platform to validate the authority the image
# More info: https://docs.openshift.com/container-platform/3.11/creating_images/guidelines.html#openshift-specific-guidelines
ENTRYPOINT ["/usr/local/bin/entrypoint"]

# Execute the operator 
CMD exec ${OPERATOR} $@`
