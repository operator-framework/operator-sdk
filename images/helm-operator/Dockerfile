FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-227

ENV HOME=/opt/helm \
    USER_NAME=helm \
    USER_UID=1001

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}
USER ${USER_UID}

ARG BIN=bin/helm-operator
COPY $BIN /usr/local/bin/helm-operator

ENTRYPOINT ["/usr/local/bin/helm-operator", "run", "--watches-file=./watches.yaml"]
