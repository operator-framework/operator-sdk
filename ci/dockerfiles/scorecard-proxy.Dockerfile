FROM osdk-builder as builder

RUN ci/tests/scaffolding/scorecard-proxy-scaffold.sh

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV PROXY=/usr/local/bin/scorecard-proxy \
    USER_UID=1001 \
    USER_NAME=proxy

# install operator binary
COPY --from=builder /scorecard/scorecard-proxy ${PROXY}

COPY --from=builder /scorecard/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
