FROM osdk-builder as builder

RUN ci/tests/e2e-go-scaffold.sh

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/memcached-operator \
    USER_UID=1001 \
    USER_NAME=memcached-operator

# install operator binary
COPY --from=builder /memcached-operator ${OPERATOR}
COPY test/test-framework/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
