# Build the operator-sdk binary
FROM --platform=$BUILDPLATFORM golang:1.24 AS builder
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
RUN GOOS=linux GOARCH=$TARGETARCH make build/operator-sdk

# Final image.
FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6

ARG TARGETARCH
RUN microdnf install -y make gcc which tar gzip

# Copy Go runtime from builder image
COPY --from=builder /usr/local/go /usr/local/go
RUN ln -sf /usr/local/go/bin/* /usr/local/bin/ && go version

COPY --from=builder /workspace/build/operator-sdk /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
