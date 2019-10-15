FROM openshift/origin-release:golang-1.13

WORKDIR /go/src/github.com/operator-framework/operator-sdk
ENV GOPROXY=https://proxy.golang.org/

COPY . .

RUN make -f ci/prow.Makefile build
