FROM quay.io/operator-framework/ansible-operator:dev

COPY ansible.cfg /etc/ansible/ansible.cfg
COPY requirements.yml ${HOME}/requirements.yml
COPY watches.yaml ${HOME}/watches.yaml

COPY roles/ ${HOME}/roles/
COPY playbooks/ ${HOME}/playbooks/
COPY inventory/ ${HOME}/inventory/
COPY plugins/ ${HOME}/plugins/
RUN ansible-galaxy collection install -r ${HOME}/requirements.yml
