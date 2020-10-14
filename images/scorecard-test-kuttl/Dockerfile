# Base image
#FROM docker.io/kudobuilder/kuttl@sha256:c9c5edc27ba6e5e994ffd8cea03368c0d4e274f3c7a00f51c1e360feb573f24e
FROM kudobuilder/kuttl:v0.6.1

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
