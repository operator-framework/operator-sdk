FROM quay.io/operator-framework/ansible-operator:dev

COPY ansible.cfg /etc/ansible/ansible.cfg
COPY . ${HOME}/

USER root

RUN chmod g+rw /etc/ansible/ansible.cfg
RUN find ${HOME} -type f -exec chmod -R g+rw {} \;  && \
    find ${HOME} -type d -exec chmod -R g+rwx {} \;

USER ${USER_UID}