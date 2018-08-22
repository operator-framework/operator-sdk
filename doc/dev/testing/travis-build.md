# TravisCI Build Information

Travis is set to run once every 24hrs against the master branch. The results of the builds can be found [here](https://travis-ci.org/operator-framework/operator-sdk/builds).

## Tool Versions
* Kubernetes: 1.10.0
* Minikube: 0.25.2
* Go: 1.10.1

## Test Workflow
1. Build the operator-sdk binary
2. Run unit tests
3. Run end-to-end tests
  - Memcached: Creates the example memcached-operator project using the operator-sdk
    - Cluster: Runs the example memcached-operator in the cluster and spins up 3 memcached containers in a deployment and verifies that all 3 are available.
    It then scales the deployment to 4 containers and verifies that there are 4 available containers in the deployment
    - Local: Same as cluster test, but runs the operator using `up local` instead of in a deployment in the cluster.
4. Ensure proper formatting
5. Ensure all go files contain a license header
6. Ensure all error messages have consistent capitalization
