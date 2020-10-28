FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN microdnf install -y golang make which
RUN microdnf install -y git

ARG BIN=operator-sdk
COPY $BIN /usr/local/bin/operator-sdk

# install kustomize
RUN git clone https://github.com/kubernetes-sigs/kustomize.git
RUN cd kustomize && \
    cd kustomize && \
    go install .
RUN ~/go/bin/kustomize version

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
