# Build the scorecard-test-chainsaw binary
FROM --platform=$BUILDPLATFORM golang:1.21 as builder
ARG TARGETARCH
ARG BUILDPLATFORM

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN GOOS=linux GOARCH=$TARGETARCH make build/scorecard-test-chainsaw

# Final image.
FROM ghcr.io/kyverno/chainsaw:v0.1.3

ENV HOME=/opt/scorecard-test-chainsaw \
    USER_NAME=scorecard-test-chainsaw \
    USER_UID=1001 \
    KUBECONFIG=

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}

COPY --from=builder /workspace/build/scorecard-test-chainsaw /usr/local/bin/scorecard-test-chainsaw
COPY --from=builder /workspace/images/scorecard-test-chainsaw/entrypoint /usr/local/bin/entrypoint

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
