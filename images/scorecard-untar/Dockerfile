FROM registry.access.redhat.com/ubi9/ubi:9.5

## Create a new non-root user to run as
ENV HOME=/opt/scorecard-untar \
    USER_NAME=scorecard-untar \
    USER_UID=1001

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}

USER ${USER_UID}
