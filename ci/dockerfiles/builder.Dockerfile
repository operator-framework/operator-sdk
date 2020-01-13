FROM openshift/origin-release:golang-1.13

WORKDIR /go/src/github.com/operator-framework/operator-sdk
ENV GOPATH=/go PATH=/go/src/github.com/operator-framework/operator-sdk/build:$PATH

COPY . .

RUN make -f ci/prow.Makefile build
