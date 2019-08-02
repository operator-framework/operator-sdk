FROM osdk-builder as builder

RUN make image/scaffold/ansible

FROM registry.access.redhat.com/ubi7/ubi

# Temporary for CI, reset /etc/passwd
RUN chmod 0644 /etc/passwd

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
RUN (yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm || true) \
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

COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/build/operator-sdk ${OPERATOR}
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/library/k8s_status.py /usr/share/ansible/openshift/
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/bin/ao-logs /usr/local/bin/ao-logs

# Ensure directory permissions are properly set
RUN mkdir -p ${HOME}/.ansible/tmp \
 && chown -R ${USER_UID}:0 ${HOME} \
 && chmod -R ug+rwx ${HOME}

ADD https://github.com/krallin/tini/releases/latest/download/tini /tini
RUN chmod +x /tini

ENTRYPOINT ["/tini", "--", "bash", "-c", "${OPERATOR} run ansible --watches-file=/opt/ansible/watches.yaml $@"]

USER ${USER_UID}
