# Base image
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
