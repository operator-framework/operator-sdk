FROM osdk-builder as builder

RUN ci/tests/e2e-helm-scaffold.sh

FROM osdk-helm

USER root

COPY --from=builder --chown=1001:0 /helm/nginx-operator/watches.yaml ${HOME}/watches.yaml
COPY --from=builder --chown=1001:0 /helm/nginx-operator/helm-charts/ ${HOME}/helm-charts

RUN find ${HOME} -type f -exec chmod -R g+rw {} \;  && \
    find ${HOME} -type d -exec chmod -R g+rwx {} \;

USER ${USER_UID}