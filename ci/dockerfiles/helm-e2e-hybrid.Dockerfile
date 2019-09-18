FROM osdk-builder as builder

RUN make image/scaffold/helm
RUN ci/tests/e2e-helm-scaffold-hybrid.sh

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/helm-operator \
    USER_UID=1001 \
    USER_NAME=helm \
    HOME=/opt/helm

COPY --from=builder /helm/nginx-operator/watches.yaml ${HOME}/watches.yaml
COPY --from=builder /helm/nginx-operator/helm-charts/ ${HOME}/helm-charts

# install operator binary
COPY --from=builder /nginx-operator ${OPERATOR}

COPY --from=builder /go/src/github.com/operator-framework/operator-sdk/bin /usr/local/bin
RUN /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
