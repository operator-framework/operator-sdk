# Base image
FROM docker.io/kudobuilder/kuttl@sha256:d23368441f313107954e80a3a5f2884f374bd5a4746193e304c7733f98d6915e
#FROM kudobuilder/kuttl:latest

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
