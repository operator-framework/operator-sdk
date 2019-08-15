// Copyright 2019 The Operator-SDK Authors
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

package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
)

//DockerfileHybrid - Dockerfile for a hybrid operator
type DockerfileHybrid struct {
	input.Input

	// Playbook - if true, include a COPY statement for playbook.yml
	Playbook bool

	// Roles - if true, include a COPY statement for the roles directory
	Roles bool

	// Watches - if true, include a COPY statement for watches.yaml
	Watches bool
}

// GetInput - gets the input
func (d *DockerfileHybrid) GetInput() (input.Input, error) {
	if d.Path == "" {
		d.Path = filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile)
	}
	d.TemplateBody = dockerFileHybridAnsibleTmpl
	d.Delims = AnsibleDelims
	return d.Input, nil
}

const dockerFileHybridAnsibleTmpl = `FROM registry.access.redhat.com/ubi7/ubi

RUN mkdir -p /etc/ansible \
    && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
    && echo '[defaults]' > /etc/ansible/ansible.cfg \
    && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
    && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

ENV OPERATOR=/usr/local/bin/ansible-operator \
    USER_UID=1001 \
    USER_NAME=ansible-operator\
    HOME=/opt/ansible

# Install python dependencies

RUN yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
 && yum install -y python-devel gcc inotify-tools \
 && easy_install pip \
 && pip install -U --no-cache-dir setuptools pip \
 && pip install --no-cache-dir --ignore-installed ipaddress \
      ansible-runner==1.2 \
      ansible-runner-http==1.0.0 \
      openshift==0.8.9 \
      ansible==2.8 \
 && yum remove -y gcc python-devel \
 && yum clean all \
 && rm -rf /var/cache/yum

COPY build/_output/bin/[[.ProjectName]] ${OPERATOR}
COPY bin /usr/local/bin
COPY library/k8s_status.py /usr/share/ansible/openshift/

RUN /usr/local/bin/user_setup

# Ensure directory permissions are properly set
RUN mkdir -p ${HOME}/.ansible/tmp \
 && chown -R ${USER_UID}:0 ${HOME} \
 && chmod -R ug+rwx ${HOME}

ADD https://github.com/krallin/tini/releases/latest/download/tini /tini
RUN chmod +x /tini

[[- if .Watches ]]
COPY watches.yaml ${HOME}/watches.yaml[[ end ]]
[[- if .Roles ]]
COPY roles/ ${HOME}/roles/[[ end ]]
[[- if .Playbook ]]
COPY playbook.yml ${HOME}/playbook.yml[[ end ]]

ENTRYPOINT ["/tini", "--", "/usr/local/bin/entrypoint"]

USER ${USER_UID}
`
