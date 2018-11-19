# Binary builder image
FROM golang:1.10-alpine3.8 AS builder

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /go/src/github.com/operator-framework/operator-sdk
COPY . /go/src/github.com/operator-framework/operator-sdk

RUN go build -o /go/bin/scorecard-proxy github.com/operator-framework/operator-sdk/test/scorecard-proxy/cmd/proxy

# Base image
FROM alpine:3.8

RUN apk upgrade --update --no-cache
USER nobody

COPY --from=builder /go/bin/scorecard-proxy /usr/local/bin/scorecard-proxy