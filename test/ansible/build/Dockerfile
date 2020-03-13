FROM quay.io/operator-framework/ansible-operator:dev

COPY ansible.cfg /etc/ansible/ansible.cfg
COPY watches.yaml ${HOME}/watches.yaml

COPY roles/ ${HOME}/roles/
COPY playbooks/ ${HOME}/playbooks/
COPY inventory/ ${HOME}/inventory/
COPY plugins/ ${HOME}/plugins/
COPY fixture_collection/ /tmp/fixture_collection/
USER root
RUN chmod -R ug+rwx /tmp/fixture_collection
USER 1001
RUN ansible-galaxy collection build /tmp/fixture_collection/ --output-path /tmp/fixture_collection/ \
 && ansible-galaxy collection install /tmp/fixture_collection/operator_sdk-test_fixtures-0.0.0.tar.gz
