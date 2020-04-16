# Base image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV TEST=/usr/local/bin/scorecard-test \
    USER_UID=1001 \
    USER_NAME=test

# install test binary 
COPY scorecard-test ${TEST}

COPY bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
