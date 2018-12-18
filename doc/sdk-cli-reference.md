# CLI Guide

```bash
Usage:
  operator-sdk [command]
```

## build

### Args

* image - is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".

### Flags
* `--enable-tests` bool - enable in-cluster testing by adding test binary to the image
* `--namespaced-manifest` string - path of namespaced resources manifest for tests (default "deploy/operator.yaml")
* `--test-location` string - location of tests (default "./test/e2e")
* `-h, --help` - help for build


### Use

The operator-sdk build command compiles the code and builds the executables. After build completes, the image is built locally in docker. Then it needs to be pushed to a remote registry.

If `--enable-tests` is set, the build command will also build the testing binary, add it to the docker image, and generate
a `deploy/test-pod.yaml` file that allows a user to run the tests as a pod on a cluster.

### Example:

#### Build

```bash
operator-sdk build quay.io/example/operator:v0.0.1

# Output:
building example-operator...

building container quay.io/example/operator:v0.0.1...
Sending build context to Docker daemon  163.9MB
Step 1/4 : FROM alpine:3.6
 ---> 77144d8c6bdc
Step 2/4 : ADD tmp/_output/bin/example-operator /usr/local/bin/example-operator
 ---> 2ada0d6ca93c
Step 3/4 : RUN adduser -D example-operator
 ---> Running in 34b4bb507c14
Removing intermediate container 34b4bb507c14
 ---> c671ec1cff03
Step 4/4 : USER example-operator
 ---> Running in bd336926317c
Removing intermediate container bd336926317c
 ---> d6b58a0fcb8c
Successfully built d6b58a0fcb8c
Successfully tagged quay.io/example/operator:v0.0.1
```

## completion

### Available Commands

#### bash - Generate bash completions

##### Flags

* `-h, --help` - help for bash

#### zsh - Generate zsh completions

##### Flags

* `-h, --help` - help for zsh

### Flags

* `-h, --help` - help for completion

### Use

Generators for shell completions

Example:

```bash
operator-sdk completion bash

# Output:
# bash completion for operator-sdk                         -*- shell-script -*-
...
# ex: ts=4 sw=4 et filetype=sh
```

## print-deps

Prints the most recent Golang packages and versions required by operators. Prints in columnar format by default.

### Flags

* `--as-file` Print packages and versions in Gopkg.toml format.

### Example

```bash
$ operator-sdk print-deps --as-file
required = [
  "k8s.io/code-generator/cmd/defaulter-gen",
  "k8s.io/code-generator/cmd/deepcopy-gen",
  "k8s.io/code-generator/cmd/conversion-gen",
  "k8s.io/code-generator/cmd/client-gen",
  "k8s.io/code-generator/cmd/lister-gen",
  "k8s.io/code-generator/cmd/informer-gen",
  "k8s.io/code-generator/cmd/openapi-gen",
  "k8s.io/gengo/args",
]

[[override]]
  name = "k8s.io/code-generator"
  revision = "6702109cc68eb6fe6350b83e14407c8d7309fd1a"
...
```

## generate

### k8s

Runs the Kubernetes [code-generators][k8s-code-generator] for all Custom Resource Definitions (CRD) apis under `pkg/apis/...`.
Currently only runs `deepcopy-gen` to generate the required `DeepCopy()` functions for all custom resource types.

**Note**: This command must be run every time the api (spec and status) for a custom resource type is updated.

#### Example

```bash
$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go

$ operator-sdk generate k8s
Running code-generation for custom resource group versions: [app:v1alpha1]
Generating deepcopy funcs

$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go
└── zz_generated.deepcopy.go
```

## olm-catalog

Parent command for all OLM Catalog-related commands.

### gen-csv

Generates a Cluster Service Version manifest file in `deploy/olm-catalog`.

#### Flags

* `--csv-version` (required) operator semantic version with which to create the CSV file.

#### Example

```bash
$ operator-sdk olm-catalog gen-csv --csv-version 0.1.1
Generating CSV manifest version 0.1.1
```

## new

Scaffolds a new operator project.

### Args

* `project-name` - name of the new project

### Flags


* `--skip-git-init` Do not init the directory as a git repository
* `--type` Type of operator to initialize: "ansible", "helm", or "go" (default "go"). Also requires the following flags if `--type=ansible` or `--type=helm`
  * `--api-version` CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
  * `--kind` CRD Kind. (e.g AppService)
* `--cluster-scoped` Initialize the operator to be cluster-scoped instead of namespace-scoped
* `-h, --help` - help for new

### Example

Go project:
```bash
mkdir $GOPATH/src/github.com/example.com/
cd $GOPATH/src/github.com/example.com/
operator-sdk new app-operator
```

Ansible project:
```bash
operator-sdk new app-operator --type=ansible --api-version=app.example.com/v1alpha1 --kind=AppService
```

Helm project:
```bash
operator-sdk new app-operator --type=helm --api-version=app.example.com/v1alpha1 --kind=AppService
```

## add

### api

Adds the
api definition for a new custom resource under `pkg/apis` and generates the CRD and CR files under `depoy/crds/...`.

#### Flags

* `--api-version` CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` CRD Kind. (e.g AppService)

#### Example

```bash
$ operator-sdk add api --api-version app.example.com/v1alpha1 --kind AppService
Create pkg/apis/app/v1alpha1/appservice_types.go
Create pkg/apis/addtoscheme_app_v1alpha1.go
Create pkg/apis/app/v1alpha1/register.go
Create pkg/apis/app/v1alpha1/doc.go
Create deploy/crds/app_v1alpha1_appservice_cr.yaml
Create deploy/crds/app_v1alpha1_appservice_crd.yaml
Running code-generation for custom resource group versions: [app:v1alpha1]
Generating deepcopy funcs
```

### controller

Adds a new
controller under `pkg/controller/<kind>/...` that, by default, reconciles a custom resource for the specified apiversion and kind.

#### Flags

* `--api-version` CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` CRD Kind. (e.g AppService)

#### Example

```bash
$ operator-sdk add controller --api-version app.example.com/v1alpha1 --kind AppService
Create pkg/controller/appservice/appservice_controller.go
Create pkg/controller/add_appservice.go
```

### crd

Generates the CRD and the CR files for the specified api-version and kind.

#### Flags

* `--api-version` CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` CRD Kind. (e.g AppService)

#### Example

```bash
$ operator-sdk add crd --api-version app.example.com/v1alpha1 --kind AppService
Generating custom resource definition (CRD) files
Create deploy/crds/app_v1alpha1_appservice_crd.yaml
Create deploy/crds/app_v1alpha1_appservice_cr.yaml
```

## test

### Available Commands

#### local
Runs the tests locally

##### Args
* <test-location> string - location of e2e test files (e.g. "./test/e2e/")

##### Flags
* `--kubeconfig` string - location of kubeconfig for kubernetes cluster (default "~/.kube/config")
* `--global-manifest` string - path to manifest for global resources (default "deploy/crd.yaml)
* `--namespaced-manifest` string - path to manifest for per-test, namespaced resources (default: combines deploy/service_account.yaml, deploy/rbac.yaml, and deploy/operator.yaml)
*  `--namespace` string - if non-empty, single namespace to run tests in (e.g. "operator-test") (default: "")
* `--go-test-flags` string - extra arguments to pass to `go test` (e.g. -f "-v -parallel=2")
* `--up-local` - enable running operator locally with go run instead of as an image in the cluster
* `--no-setup` - disable test resource creation
* `--image` string - use a different operator image from the one specified in the namespaced manifest
* `-h, --help` - help for local

##### Use

The operator-sdk test command runs go tests built using the Operator SDK's test framework.

##### Example:

```bash
$ operator-sdk test local ./test/e2e/

# Output:
ok  	github.com/operator-framework/operator-sdk-samples/memcached-operator/test/e2e	20.410s
```

#### cluster
Runs the e2e tests packaged in an operator image as a pod in the cluster

##### Args
* <image-name> string - the operator image that is used to run the tests in a pod (e.g. "quay.io/example/memcached-operator:v0.0.1")

##### Flags
* `--kubeconfig` string - location of kubeconfig for kubernetes cluster (default "~/.kube/config")
* `--image-pull-policy` string - set test pod image pull policy. Allowed values: Always, Never (default "Always")
* `--namespace` string - namespace to run tests in (default "default")
* `--pending-timeout` int - timeout in seconds for testing pod to stay in pending state (default 60s)
* `--service-account` string - service account to run tests on (default "default")
* `--help` - help for cluster

##### Use

The operator-sdk test command runs go tests embedded in an operator image built using the Operator SDK.

##### Example:

```bash
$ operator-sdk test cluster quay.io/example/memcached-operator:v0.0.1

# Output:
Test Successfully Completed
```

## up

### Available Commands

#### local - Launches the operator locally

##### Use

The operator-sdk up local command launches the operator on the local machine
by building the operator binary with the ability to access a
kubernetes cluster using a kubeconfig file.

##### Flags

* `--kubeconfig` string - The file path to kubernetes configuration file; defaults to $HOME/.kube/config

* `--namespace` string - The namespace where the operator watches for changes. (default "default")

* `--operator-flags` - Flags that the local operator may need.

* `-h, --help` - help for local

##### Example

```bash
operator-sdk up local --kubeconfig "mycluster.kubecfg" \
  --namespace "default" \
  --operator-flags "--flag1 value1 --flag2=value2"
```

The below example will use the default kubeconfig, the default namespace environment var, and pass in flags for the operator.
To use the operator flags, your operator must know how to handle the option. Below imagine an operator that understands the `resync-interval` flag.

```bash
operator-sdk up local --operator-flags "--resync-interval 10"
```

If you are planning on using a different namespace than the default, then you should use the `--namespace` flag to change where the operator is watching for custom resources to be created.
For this to work your operator must handle the `WATCH_NAMESPACE` environment variable. To do that you can use the [utility function][utility_link] `k8sutil.GetWatchNamespace` in your operator.

```bash
operator-sdk up local --namespace "testing"
```

### Flags

* `-h, --help` - help for up

[utility_link]: https://github.com/operator-framework/operator-sdk/blob/89bf021063d18b6769bdc551ed08fc37027939d5/pkg/util/k8sutil/k8sutil.go#L140
[k8s-code-generator]: https://github.com/kubernetes/code-generator
