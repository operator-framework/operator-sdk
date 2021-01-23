FROM registry.access.redhat.com/ubi8/ubi:8.3-227

RUN mkdir -p /etc/ansible \
  && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
  && echo '[defaults]' > /etc/ansible/ansible.cfg \
  && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
  && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

ENV HOME=/opt/ansible \
    USER_NAME=ansible \
    USER_UID=1001

# Install python dependencies
# Ensure fresh metadata rather than cached metadata in the base by running
# yum clean all && rm -rf /var/yum/cache/* first
COPY Pipfile* .
ENV PIP_NO_CACHE_DIR=1 \
    PIPENV_SYSTEM=1 \
    PIPENV_CLEAR=1
RUN INSTALL_DEPS="libffi-devel openssl-devel python38-devel gcc" \
  && yum clean all && rm -rf /var/cache/yum/* \
  && yum -y update \
  && yum install -y ${INSTALL_DEPS} python38-pip \
  && pip3 install pipenv==2020.11.15 \
  && pipenv install --deploy --system \
  && pipenv check \
  && yum remove -y ${INSTALL_DEPS} \
  && yum clean all \
  && rm -rf /var/cache/yum

# Ensure directory permissions are properly set
RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd \
  && mkdir -p ${HOME}/.ansible/tmp \
  && chown -R ${USER_UID}:0 ${HOME} \
  && chmod -R ug+rwx ${HOME}

RUN TINIARCH=$(case $(arch) in x86_64) echo -n amd64 ;; ppc64le) echo -n ppc64el ;; aarch64) echo -n arm64 ;; *) echo -n $(arch) ;; esac) \
  && curl -L -o /tini https://github.com/krallin/tini/releases/latest/download/tini-$TINIARCH \
  && chmod +x /tini

WORKDIR ${HOME}
USER ${USER_UID}

ARG BIN=bin/ansible-operator
COPY $BIN /usr/local/bin/ansible-operator

ENTRYPOINT ["/tini", "--", "/usr/local/bin/ansible-operator", "run", "--watches-file=./watches.yaml"]
