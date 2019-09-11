FROM osdk-builder as builder

RUN make image/scaffold/ansible
RUN ci/tests/e2e-ansible-scaffold-hybrid.sh

FROM registry.access.redhat.com/ubi8/ubi:latest

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
# Ensure fresh metadata rather than cached metadata in the base by running
# sed -i 's|enabled=1|enabled=0|g' /etc/yum/pluginconf.d/product-id.conf && rm -rf /var/yum/cache/* first
RUN sed -i 's|enabled=1|enabled=0|g' /etc/yum/pluginconf.d/product-id.conf && rm -rf /var/cache/yum/* \
 && (yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm || true) \
 && yum install -y python36-devel.x86_64 gcc \
 && pip3 install --user --no-cache-dir --ignore-installed ipaddress \
      ansible-runner==1.2 \
      ansible-runner-http==1.0.0 \
      openshift==0.8.9 \
      ansible==2.8 \
 && yum remove -y gcc python36-devel.x86_64 \
 && yum clean all \
 && rm -rf /var/cache/yum

# install operator binary
COPY --from=builder /memcached-operator ${OPERATOR}
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/library/k8s_status.py /usr/share/ansible/openshift/
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/bin/ao-logs /usr/local/bin/ao-logs
COPY --from=builder /ansible/memcached-operator/watches.yaml ${HOME}/watches.yaml
COPY --from=builder /ansible/memcached-operator/roles/ ${HOME}/roles/

# Ensure directory permissions are properly set
RUN mkdir -p ${HOME}/.ansible/tmp \
 && chown -R ${USER_UID}:0 ${HOME} \
 && chmod -R ug+rwx ${HOME}

ADD https://github.com/krallin/tini/releases/latest/download/tini /tini
RUN chmod +x /tini

ENTRYPOINT ["/tini", "--", "bash", "-c", "${OPERATOR} run ansible --watches-file=/opt/ansible/watches.yaml $@"]

USER ${USER_UID}
