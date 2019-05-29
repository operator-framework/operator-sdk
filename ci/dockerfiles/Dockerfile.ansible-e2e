FROM osdk-builder as builder

RUN ci/tests/e2e-ansible-scaffold.sh

FROM osdk-ansible

COPY --from=builder /ansible/memcached-operator/watches.yaml ${HOME}/watches.yaml

COPY --from=builder /ansible/memcached-operator/roles/ ${HOME}/roles/
