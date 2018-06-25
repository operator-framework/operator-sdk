# TravisCI Build Information

Travis is set to run once every 24hrs against the master branch. The results of the builds can be found [here](https://travis-ci.org/operator-framework/operator-sdk/builds).

## Tool Versions
* Kubernetes: 1.9.0
* Minikube: 0.25.2
* Go: 1.10.1

## Test Workflow
1. Run unit tests
2. Ensure proper formatting
3. Install the sdk
4. Create memcached operator from [user-guide.md](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md#build-and-run-the-operator)
5. Fill out handler.go and types.go
6. Generate and build the operator for k8s
7. Run the deployment