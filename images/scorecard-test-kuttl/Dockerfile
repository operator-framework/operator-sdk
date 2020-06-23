# Base image
#FROM docker.io/kuttl@sha256:38a7be070c4a8ed7bf96b20f211f506af4015527de6136f81e1a34c7959a558d
FROM kuttl:latest

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
