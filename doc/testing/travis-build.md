# TravisCI Build Information

## Tool Versions
* Kubernetes: 1.9.0
* Minikube: 0.25.2
* Go: 1.10.1

## Test Workflow
1. Run unit tests
2. Ensure proper formatting
3. Install the sdk
4. Create memcached operator
5. Fill out handler.go and types.go
6. Generate and build the operator for k8s
7. Run the deployment