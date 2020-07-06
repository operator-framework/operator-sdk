# Base image
FROM docker.io/kudobuilder/kuttl@sha256:970fcac6cf2eeba433a9e00eb9e24a7ecb0a313c2f1d0cc120cfd206210cd0ba
#FROM kuttl:latest

ENV TESTKUTTL=/usr/local/bin/scorecard-test-kuttl \
    USER_UID=1001 \
    USER_NAME=test \
    KUBECONFIG=

# install test binary 
COPY scorecard-test-kuttl ${TESTKUTTL}

COPY bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
