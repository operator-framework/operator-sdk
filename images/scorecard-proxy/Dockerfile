# Base image
FROM alpine:3.8

ENV PROXY=/usr/local/bin/scorecard-proxy \
    USER_UID=1001 \
    USER_NAME=proxy

# install operator binary
COPY scorecard-proxy ${PROXY}

COPY bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
