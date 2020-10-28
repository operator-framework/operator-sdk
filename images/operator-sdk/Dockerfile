FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN microdnf install -y golang make which

ARG BIN=bin/operator-sdk
COPY $BIN /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
