FROM openshift/origin-release:golang-1.13

WORKDIR /go/src/github.com/operator-framework/operator-sdk
ENV GOPATH=/go PATH=/go/src/github.com/operator-framework/operator-sdk/build:$PATH GOPROXY=https://proxy.golang.org/ GO111MODULE=on

COPY . .

RUN make -f ci/prow.Makefile build
