FROM quay.io/operator-framework/ansible-operator:dev

COPY ansible.cfg /etc/ansible/ansible.cfg
COPY watches.yaml ${HOME}/watches.yaml

COPY roles/ ${HOME}/roles/
COPY playbooks/ ${HOME}/playbooks/
COPY inventory/ ${HOME}/inventory/
COPY plugins/ ${HOME}/plugins/
