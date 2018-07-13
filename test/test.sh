#!/bin/bash
mkdir -p $GOPATH/src/github.com/example-inc

go test ./...
go vet ./...
make install
cd test
go build
dep ensure
cp test $GOPATH/src/github.com/example-inc

cd $GOPATH/src/github.com/example-inc
operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached
cd memcached-operator
rm -rf vendor/github.com/operator-framework/operator-sdk/pkg
ln -s ${TRAVIS_BUILD_DIR}/pkg vendor/github.com/operator-framework/operator-sdk/pkg
curl https://raw.githubusercontent.com/operator-framework/operator-sdk/master/example/memcached-operator/handler.go.tmpl -o pkg/stub/handler.go
head -n -6 pkg/apis/cache/v1alpha1/types.go > tmp.txt
mv tmp.txt pkg/apis/cache/v1alpha1/types.go
echo 'type MemcachedSpec struct {	Size int32 `json:"size"`}' >> pkg/apis/cache/v1alpha1/types.go
echo 'type MemcachedStatus struct {Nodes []string `json:"nodes"`}' >> pkg/apis/cache/v1alpha1/types.go
operator-sdk generate k8s
operator-sdk build quay.io/example/memcached-operator:v0.0.1
sed -ie 's/imagePullPolicy: Always/imagePullPolicy: Never/g' deploy/operator.yaml
../test

# Cleanup
kubectl delete -f deploy/cr.yaml
kubectl delete -f deploy/operator.yaml
