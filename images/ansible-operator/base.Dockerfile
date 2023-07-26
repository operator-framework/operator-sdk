# This Dockerfile defines the base image for the ansible-operator image.
# It is built with dependencies that take a while to download, thus speeding
# up ansible deploy jobs.

FROM registry.access.redhat.com/ubi8/ubi:8.8 AS builder

# Install Rust so that we can ensure backwards compatibility with installing/building the cryptography wheel across all platforms
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"
RUN rustc --version

# When cross-compiling the container, cargo uncontrollably consumes memory and
# gets killed by the OOM Killer when it fetches dependencies. The workaround is
# to use the git executable.
# See https://github.com/rust-lang/cargo/issues/10583 for details.
ENV CARGO_NET_GIT_FETCH_WITH_CLI=true

# Copy python dependencies (including ansible) to be installed using Pipenv
COPY Pipfile* ./
# Instruct pip(env) not to keep a cache of installed packages,
# to install into the global site-packages and
# to clear the pipenv cache as well
ENV PIP_NO_CACHE_DIR=1 \
    PIPENV_SYSTEM=1 \
    PIPENV_CLEAR=1
# Ensure fresh metadata rather than cached metadata, install system and pip python deps,
# and remove those not needed at runtime.
# pip3~=21.1 fixes a vulnerability described in https://github.com/pypa/pip/pull/9827.
RUN set -e && yum clean all && rm -rf /var/cache/yum/* \
  && yum update -y \
  && yum install -y libffi-devel openssl-devel python39-devel gcc python39-pip python39-setuptools \
  && pip3 install --upgrade pip~=23.1.2 \
  && pip3 install pipenv==2023.6.26 \
  && pipenv install --deploy \
  && pipenv check \
  && yum remove -y gcc libffi-devel openssl-devel python39-devel \
  && yum clean all \
  && rm -rf /var/cache/yum

FROM registry.access.redhat.com/ubi8/ubi:8.8
ARG TARGETARCH

# Label this image with the repo and commit that built it, for freshmaking purposes.
ARG GIT_COMMIT=devel
LABEL git_commit=$GIT_COMMIT

RUN mkdir -p /etc/ansible \
  && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
  && echo '[defaults]' > /etc/ansible/ansible.cfg \
  && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
  && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

RUN set -e && yum clean all && rm -rf /var/cache/yum/* \
  && yum update -y \
  && yum install -y python39-pip python39-setuptools \
  && pip3 install --upgrade pip~=23.1.2 \
  && pip3 install pipenv==2023.6.26 \
  && yum clean all \
  && rm -rf /var/cache/yum

COPY --from=builder /usr/local/lib64/python3.9/site-packages /usr/local/lib64/python3.9/site-packages
COPY --from=builder /usr/local/lib/python3.9/site-packages /usr/local/lib/python3.9/site-packages

ENV TINI_VERSION=v0.19.0
RUN curl -L -o /tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-${TARGETARCH} \
  && chmod +x /tini && /tini --version