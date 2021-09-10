# Build the operator-sdk binary
FROM --platform=$BUILDPLATFORM golang:1.17.1 as builder
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
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

ENV GO_VERSION 1.16.4

ARG TARGETARCH
RUN microdnf install -y make which tar gzip
RUN curl -sSLo /tmp/go.tar.gz https://golang.org/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz \
	&& rm -rf /usr/local/go \
	&& tar -C /usr/local -xzf /tmp/go.tar.gz \
	&& ln -sf /usr/local/go/bin/* /usr/local/bin/ \
	&& rm -f /tmp/go.tar.gz \
	&& go version

COPY --from=builder /workspace/build/operator-sdk /usr/local/bin/operator-sdk

ENTRYPOINT ["/usr/local/bin/operator-sdk"]
