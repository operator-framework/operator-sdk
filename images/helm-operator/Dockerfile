# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.17.0 as builder
ARG TARGETARCH

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
RUN GOOS=linux GOARCH=$TARGETARCH make build/helm-operator

# Final image.
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

ENV HOME=/opt/helm \
    USER_NAME=helm \
    USER_UID=1001

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}
USER ${USER_UID}

COPY --from=builder /workspace/build/helm-operator /usr/local/bin/helm-operator

ENTRYPOINT ["/usr/local/bin/helm-operator", "run", "--watches-file=./watches.yaml"]
