FROM osdk-builder as builder

RUN make image/scaffold/helm
RUN ci/tests/e2e-helm-scaffold-hybrid.sh

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/helm-operator \
    USER_UID=1001 \
    USER_NAME=helm \
    HOME=/opt/helm

COPY --from=builder --chown=1001:0 /helm/nginx-operator/watches.yaml ${HOME}/watches.yaml
COPY --from=builder --chown=1001:0 /helm/nginx-operator/helm-charts/ ${HOME}/helm-charts

RUN find ${HOME} -type f -exec chmod -R g+rw {} \;  && \
    find ${HOME} -type d -exec chmod -R g+rwx {} \;

# install operator binary
COPY --from=builder /nginx-operator ${OPERATOR}

COPY --from=builder --chown=1001:0 /go/src/github.com/operator-framework/operator-sdk/bin /usr/local/bin
RUN chmod -R g+rwx /usr/local/bin && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
