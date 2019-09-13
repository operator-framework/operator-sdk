FROM osdk-builder as builder

RUN make image/scaffold/helm

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/helm-operator \
    USER_UID=1001 \
    USER_NAME=helm \
    HOME=/opt/helm

# install operator binary
COPY --from=builder --chown=1001:0 /go/src/github.com/operator-framework/operator-sdk/build/operator-sdk ${OPERATOR}

COPY --from=builder --chown=1001:0 /go/src/github.com/operator-framework/operator-sdk/bin /usr/local/bin
RUN chmod -R g+rwx /usr/local/bin && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
