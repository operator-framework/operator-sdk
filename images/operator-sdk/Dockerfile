# Build the operator-sdk binary
FROM --platform=$BUILDPLATFORM golang:1.15 as builder
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
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3

RUN microdnf install -y golang make which

COPY --from=builder /workspace/build/operator-sdk /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
