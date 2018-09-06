# E2E Testing the Operator SDK
The operator-sdk is automatically tested using both unit tests and e2e tests anytime
a pull request is made. The e2e tests ensure that the operator-sdk acts as intended by
simulating how a typical user might use the SDK. The automated tests test each PR and run in
Travis CI, and Travis CI has a couple of features to simplify the e2e tests that we run.
However, it is possible to run the e2e tests locally as well.

## Running the E2E Tests Locally
To run the tests locally, the tests either need access to a remote kubernetes cluster or a
local minikube instance running on the machine.

### Remote Kubernetes Instance
To run the tests on a remote cluster, the tests need access to a remote kubernetes cluster
running kubernetes 1.10 as well as a docker image repo to push the operator image to,
such as quay.io. To run the test, use this command:
```
$ go test ./test/e2e/... -kubeconfig "path-to-config" -image "<repository>:<tag>"
```

This will run the tests on the cluster specified by the provided kubeconfig and the
memcached-operator image that is built will be pushed to `<repository>:<tag>`.

### Local Minikube Instance
To run the e2e tests on a local minikube cluster, the minikube instance must be
started and the host's docker client must be linked to the minikube instance's docker daemon,
which allows the host to add images to the minikube's local image registry directly.
To do this, run these commands:
```
$ minikube start --kubernetes-version v1.10.0
$ eval $(minikube docker-env)
```

Once that is complete, the test can be run with this command:
```
$ go test ./test/e2e/...
```

The test will run using the kube config in $HOME/.kube/config (which is where the minikube
kubeconfig is placed by default) and the operator image will be built and stored on the
minikube instance's local image registry.

## Cleanup of the E2E Tests
The e2e tests create a new project using the operator-sdk to run in the provided
cluster. The tests are designed to cleanup everything that gets created, but some errors
can cause these cleanups to fail. For example, if a segfault occurs or a user kills the
testing process, the cleanup functions will not run. To manually clean up a test:
1. Delete the created project in $GOPATH/src/github.com/example-inc/memcached-operator
2. Delete the namespaces that the tests run in, which also deletes any resources created
within the namespaces. The namespaces start with `memcached-memcached-group`.
3. Delete the CRD (`kubectl delete -f deploy/crd.yaml`).
