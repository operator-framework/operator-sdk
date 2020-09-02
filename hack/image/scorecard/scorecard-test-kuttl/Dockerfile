FROM kudobuilder/kuttl:v0.5.2

ENV TESTKUTTL=/usr/local/bin/scorecard-test-kuttl \
    USER_UID=1001 \
    USER_NAME=test \
    HOME=/opt/test \
    KUBECONFIG=""

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd
COPY scorecard-test-kuttl-dev-linux-gnu /usr/local/bin/scorecard-test-kuttl

WORKDIR ${HOME}
USER ${USER_UID}
ENTRYPOINT ["/usr/local/bin/entrypoint"]
