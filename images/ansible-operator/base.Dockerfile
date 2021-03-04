# This Dockerfile defines the base image for the ansible-operator image.
# It is built with dependencies that take a while to download, thus speeding
# up ansible deploy jobs.

FROM registry.access.redhat.com/ubi8/ubi:8.3-227
ARG TARGETARCH

RUN mkdir -p /etc/ansible \
  && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
  && echo '[defaults]' > /etc/ansible/ansible.cfg \
  && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
  && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

# Install python dependencies
# Ensure fresh metadata rather than cached metadata in the base by running
# yum clean all && rm -rf /var/yum/cache/* first
RUN yum clean all && rm -rf /var/cache/yum/* \
  && yum -y update \
  && yum install -y libffi-devel openssl-devel python38-devel gcc python38-pip python38-setuptools \
  && pip3 install --no-cache-dir \
    cryptography==3.3.2 \
    ansible-runner==1.3.4 \
    ansible-runner-http==1.0.0 \
    ipaddress==1.0.23 \
    kubernetes==10.1.0 \
    openshift==0.10.3 \
    ansible==2.9.15 \
    jmespath==0.10.0 \
  && yum remove -y gcc libffi-devel openssl-devel python38-devel \
  && yum clean all \
  && rm -rf /var/cache/yum

ENV TINI_VERSION=v0.19.0
RUN curl -L -o /tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-${TARGETARCH} \
  && chmod +x /tini && /tini --version
