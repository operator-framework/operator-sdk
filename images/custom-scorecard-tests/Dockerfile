# Build the custom-scorecard-tests binary
FROM --platform=$BUILDPLATFORM golang:1.21 as builder
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
RUN GOOS=linux GOARCH=$TARGETARCH make build/custom-scorecard-tests

# Final image.
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10

ENV HOME=/opt/custom-scorecard-tests \
    USER_NAME=custom-scorecard-tests \
    USER_UID=1001

RUN echo "${USER_NAME}:x:${USER_UID}:0:${USER_NAME} user:${HOME}:/sbin/nologin" >> /etc/passwd

WORKDIR ${HOME}

COPY --from=builder /workspace/build/custom-scorecard-tests /usr/local/bin/custom-scorecard-tests

ENTRYPOINT ["/usr/local/bin/custom-scorecard-tests"]

USER ${USER_UID}
