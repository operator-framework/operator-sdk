#FROM docker.io/kudobuilder/kuttl@sha256:c9c5edc27ba6e5e994ffd8cea03368c0d4e274f3c7a00f51c1e360feb573f24e
FROM kudobuilder/kuttl:v0.7.1

ENV HOME=/opt/scorecard-test-kuttl \
    USER_NAME=scorecard-test-kuttl \
    USER_UID=1001 \
    KUBECONFIG=

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}

ARG BIN=bin/scorecard-test-kuttl
COPY $BIN /usr/local/bin/scorecard-test-kuttl
COPY entrypoint /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
