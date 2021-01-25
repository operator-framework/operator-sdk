FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN microdnf install -y golang make which

ARG BIN=bin/operator-sdk
COPY $BIN /usr/local/bin/operator-sdk

RUN GOBIN=/usr/local/bin/ GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 \
 && GOBIN=/usr/local/bin/ GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
