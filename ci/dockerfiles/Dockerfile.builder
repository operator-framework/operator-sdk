FROM openshift/origin-release:golang-1.12

WORKDIR /go/src/github.com/operator-framework/operator-sdk
# Set gopath before build and include build destination in PATH
ENV GOPATH=/go PATH=/go/src/github.com/operator-framework/operator-sdk/build:$PATH

COPY . .

RUN make ci-build
