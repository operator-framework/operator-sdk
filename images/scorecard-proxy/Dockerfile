# Base image
FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV PROXY=/usr/local/bin/scorecard-proxy \
    USER_UID=1001 \
    USER_NAME=proxy

# install operator binary
COPY scorecard-proxy ${PROXY}

COPY bin /usr/local/bin
RUN chmod -R g+rwx /usr/local/bin && \
    /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
