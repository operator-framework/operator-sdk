FROM osdk-builder as builder

RUN make image/scaffold/ansible

FROM ansible/ansible-runner:1.2

RUN yum remove -y ansible python-idna
RUN yum install -y inotify-tools && yum clean all
RUN pip uninstall ansible-runner -y

RUN pip install --upgrade setuptools==41.0.1
RUN pip install "urllib3>=1.23,<1.25"
RUN pip install ansible==2.7.10 \
	ansible-runner==1.2 \
	ansible-runner-http==1.0.0 \
	idna==2.7 \
	"kubernetes>=8.0.0,<9.0.0" \
	openshift==0.8.8

RUN mkdir -p /etc/ansible \
    && echo "localhost ansible_connection=local" > /etc/ansible/hosts \
    && echo '[defaults]' > /etc/ansible/ansible.cfg \
    && echo 'roles_path = /opt/ansible/roles' >> /etc/ansible/ansible.cfg \
    && echo 'library = /usr/share/ansible/openshift' >> /etc/ansible/ansible.cfg

ENV OPERATOR=/usr/local/bin/ansible-operator \
    USER_UID=1001 \
    USER_NAME=ansible-operator\
    HOME=/opt/ansible

# install operator binary
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/build/operator-sdk ${OPERATOR}
# install k8s_status Ansible Module
COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/library/k8s_status.py /usr/share/ansible/openshift/

COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/bin /usr/local/bin
RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
