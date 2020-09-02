FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV TEST=/usr/local/bin/custom-scorecard-tests \
    USER_UID=1001 \
    USER_NAME=test \
    HOME=/opt/test

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd
COPY custom-scorecard-test-dev-linux-gnu /usr/local/bin/custom-scorecard-test

WORKDIR ${HOME}
USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/entrypoint"]
