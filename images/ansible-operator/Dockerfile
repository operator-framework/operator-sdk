# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.15 as builder
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN GOOS=linux GOARCH=$TARGETARCH make build/ansible-operator

# Final image.
FROM registry.access.redhat.com/ubi8/ubi:8.3-227
ARG TARGETARCH

RUN mkdir -p /etc/ansible \
  && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
  && echo '[defaults]' > /etc/ansible/ansible.cfg \
  && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
  && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

ENV HOME=/opt/ansible \
    USER_NAME=ansible \
    USER_UID=1001

# Install python dependencies after freshening metadata.
RUN dnf clean all && rm -rf /var/cache/dnf/* \
  && dnf update -y \
  && dnf install -y libffi-devel openssl-devel python38-devel gcc python38-pip python38-setuptools \
  && pip3 install --no-cache-dir \
    cryptography==3.3.2 \
    ansible-runner==1.3.4 \
    ansible-runner-http==1.0.0 \
    ipaddress==1.0.23 \
    kubernetes==10.1.0 \
    openshift==0.10.3 \
    ansible==2.9.15 \
    jmespath==0.10.0 \
  && dnf remove -y gcc libffi-devel openssl-devel python38-devel \
  && dnf clean all && rm -rf /var/cache/dnf/*

# Ensure directory permissions are properly set
RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd \
  && mkdir -p ${HOME}/.ansible/tmp \
  && chown -R ${USER_UID}:0 ${HOME} \
  && chmod -R ug+rwx ${HOME}

RUN curl -L -o /tini https://github.com/krallin/tini/releases/latest/download/tini-${TARGETARCH} \
  && chmod +x /tini

WORKDIR ${HOME}
USER ${USER_UID}

COPY --from=builder /workspace/build/ansible-operator /usr/local/bin/ansible-operator

ENTRYPOINT ["/tini", "--", "/usr/local/bin/ansible-operator", "run", "--watches-file=./watches.yaml"]
