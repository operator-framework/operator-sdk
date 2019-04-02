# CLI Guide

```bash
Usage:
  operator-sdk [command]
```

## build

### Args

* `image` - is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".

### Flags

* `--enable-tests` - enable in-cluster testing by adding test binary to the image
* `--namespaced-manifest` string - path of namespaced resources manifest for tests (default "deploy/operator.yaml")
* `--test-location` string - location of tests (default "./test/e2e")
* `--docker-build-args` string - extra, optional docker build arguments as one string such as `"--build-arg https_proxy=$https_proxy"` (default "")
* `-h, --help` - help for build

### Use

The operator-sdk build command compiles the code and builds the executables. After build completes, the image is built locally in docker. Then it needs to be pushed to a remote registry.

If `--enable-tests` is set, the build command will also build the testing binary, add it to the docker image, and generate
a `deploy/test-pod.yaml` file that allows a user to run the tests as a pod on a cluster.

### Example

#### Build

```console
$ operator-sdk build quay.io/example/operator:v0.0.1
building example-operator...

building container quay.io/example/operator:v0.0.1...
Sending build context to Docker daemon  163.9MB
Step 1/4 : FROM registry.access.redhat.com/ubi7-dev-preview/ubi-minimal:7.6
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

```console
$ operator-sdk completion bash
# bash completion for operator-sdk                         -*- shell-script -*-
...
# ex: ts=4 sw=4 et filetype=sh
```

## print-deps

Prints the most recent Golang packages and versions required by operators. Prints in columnar format by default.

### Flags

* `--as-file` - Print packages and versions in Gopkg.toml format.

### Example

```console
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

### Flags

* `--header-file` string - Path to file containing headers for generated files (optional).

#### Example

```console
$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go

$ operator-sdk generate k8s
INFO[0000] Running deepcopy code-generation for Custom Resource group versions: [app:[v1alpha1], ]
INFO[0001] Code-generation complete.                    

$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go
└── zz_generated.deepcopy.go
```

### openapi

Runs the [kube-openapi][openapi-code-generator] OpenAPIv3 code generator for all Custom Resource Definition (CRD) API tagged fields under `pkg/apis/...`.

**Note**: This command must be run every time a tagged API struct or struct field for a custom resource type is updated.

#### Example

```console
$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go

$ operator-sdk generate openapi
INFO[0000] Running OpenAPI code-generation for Custom Resource group versions: [app:[v1alpha1], ]
INFO[0001] Created deploy/crds/app_v1alpha1_appservice_crd.yaml
INFO[0001] Code-generation complete.                    

$ tree pkg/apis/app/v1alpha1/
pkg/apis/app/v1alpha1/
├── appservice_types.go
├── doc.go
├── register.go
└── zz_generated.openapi.go
```

## olm-catalog

Parent command for all OLM Catalog related commands.

### gen-csv

Writes a Cluster Service Version (CSV) manifest and optionally CRD files to `deploy/olm-catalog/{operator-name}/{csv-version}`.

#### Flags

* `--csv-version` string - (required) Semantic version of the CSV manifest.
* `--from-version` string - Semantic version of CSV manifest to use as a base for a new version.
* `--csv-config` string - Path to CSV config file. Defaults to deploy/olm-catalog/csv-config.yaml.
* `--update-crds` Update CRD manifests in deploy/{operator-name}/{csv-version} using the latest CRD manifests.

#### Example

```console
$ operator-sdk olm-catalog gen-csv --csv-version 0.1.0 --update-crds
INFO[0000] Generating CSV manifest version 0.1.0
INFO[0000] Fill in the following required fields in file deploy/olm-catalog/operator-name/0.1.0/operator-name.v0.1.0.clusterserviceversion.yaml:
	spec.keywords
	spec.maintainers
	spec.provider
	spec.labels
INFO[0000] Created deploy/olm-catalog/operator-name/0.1.0/operator-name.v0.1.0.clusterserviceversion.yaml     
```

## migrate

Adds a main.go source file and any associated source files for an operator that
is not of the "go" type.

**Note**: This command will look for playbook.yml in the project root, if you use the .yaml extension
you will need to rename it before running migrate or manually add it to your Dockerfile.

### Example

```console
$ operator-sdk migrate
2019/01/10 15:02:45 No playbook was found, so not including it in the new Dockerfile
2019/01/10 15:02:45 renamed Dockerfile to build/Dockerfile.sdkold and replaced with newer version
2019/01/10 15:02:45 Compare the new Dockerfile to your old one and manually migrate any customizations
INFO[0000] Created cmd/manager/main.go
INFO[0000] Created Gopkg.toml
INFO[0000] Created build/Dockerfile
INFO[0000] Created bin/entrypoint
INFO[0000] Created bin/user_setup
```

## new

Scaffolds a new operator project.

### Args

* `project-name` - name of the new project

### Flags

* `--skip-git-init` - Do not init the directory as a git repository
* `--type` string - Type of operator to initialize: "ansible", "helm", or "go" (default "go"). Also requires the following flags if `--type=ansible` or `--type=helm`
* `--api-version` string - CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` string - CRD Kind. (e.g AppService)
* `--generate-playbook` - Generate a playbook skeleton. (Only used for `--type ansible`)
* `--cluster-scoped` - Initialize the operator to be cluster-scoped instead of namespace-scoped
* `--helm-chart` string - Initialize helm operator with existing helm chart (`<URL>`, `<repo>/<name>`, or local path)
* `--helm-chart-repo` string - Chart repository URL for the requested helm chart
* `--helm-chart-version` string - Specific version of the helm chart (default is latest version)
* `-h, --help` - help for new

### Example

#### Go project

```console
$ mkdir $GOPATH/src/github.com/example.com/
$ cd $GOPATH/src/github.com/example.com/
$ operator-sdk new app-operator
```

#### Ansible project

```console
$ operator-sdk new app-operator --type=ansible --api-version=app.example.com/v1alpha1 --kind=AppService
```

#### Helm project

For more details about creating new Helm operator projects, see the [Helm user guide][helm-user-guide-create-project].

```console
$ operator-sdk new app-operator --type=helm \
    --api-version=app.example.com/v1alpha1 \
    --kind=AppService

$ operator-sdk new app-operator --type=helm \
    --api-version=app.example.com/v1alpha1 \
    --kind=AppService \
    --helm-chart=myrepo/app

$ operator-sdk new app-operator --type=helm \
    --helm-chart=myrepo/app

$ operator-sdk new app-operator --type=helm \
    --helm-chart=myrepo/app \
    --helm-chart-version=1.2.3

$ operator-sdk new app-operator --type=helm \
    --helm-chart=app \
    --helm-chart-repo=https://charts.mycompany.com/

$ operator-sdk new app-operator --type=helm \
    --helm-chart=app \
    --helm-chart-repo=https://charts.mycompany.com/ \
    --helm-chart-version=1.2.3

$ operator-sdk new app-operator --type=helm \
    --helm-chart=/path/to/local/chart-directories/app/

$ operator-sdk new app-operator --type=helm \
    --helm-chart=/path/to/local/chart-archives/app-1.2.3.tgz
```

## add

### api

Adds the API definition for a new custom resource under `pkg/apis` and generates the CRD and CR files under `depoy/crds/...`, and generates Kubernetes deepcopy functions and OpenAPIv3 validation specs for the new API.

#### Flags

* `--api-version` string - CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` string - CRD Kind. (e.g AppService)
* `--header-file` string - Path to file containing headers for generated files (optional).

#### Example

```console
$ operator-sdk add api --api-version app.example.com/v1alpha1 --kind AppService
INFO[0000] Generating api version app.example.com/v1alpha1 for kind AppService.
INFO[0000] Created pkg/apis/app/v1alpha1/appservice_types.go
INFO[0000] Created pkg/apis/addtoscheme_app_v1alpha1.go  
INFO[0000] Created pkg/apis/app/v1alpha1/register.go     
INFO[0000] Created pkg/apis/app/v1alpha1/doc.go          
INFO[0000] Created deploy/crds/app_v1alpha1_appservice_cr.yaml
INFO[0000] Created deploy/crds/app_v1alpha1_appservice_crd.yaml
INFO[0001] Running deepcopy code-generation for Custom Resource group versions: [app:[v1alpha1], ]
INFO[0002] Code-generation complete.                    
INFO[0002] Running OpenAPI code-generation for Custom Resource group versions: [app:[v1alpha1], ]
INFO[0004] Created deploy/crds/app_v1alpha1_appservice_crd.yaml
INFO[0004] Code-generation complete.                    
INFO[0004] API generation complete.
```

### controller

Adds a new controller under `pkg/controller/<kind>/...` that, by default, reconciles a custom resource for the specified apiversion and kind.

#### Flags

* `--api-version` string - CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` string - CRD Kind. (e.g AppService)

#### Example

```console
$ operator-sdk add controller --api-version app.example.com/v1alpha1 --kind AppService
Created pkg/controller/appservice/appservice_controller.go
Created pkg/controller/add_appservice.go
```

### crd

Generates the CRD and the CR files for the specified api-version and kind.

#### Flags

* `--api-version` string - CRD APIVersion in the format `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` string - CRD Kind. (e.g AppService)

#### Example

```console
$ operator-sdk add crd --api-version app.example.com/v1alpha1 --kind AppService
Generating custom resource definition (CRD) files
Created deploy/crds/app_v1alpha1_appservice_crd.yaml
Created deploy/crds/app_v1alpha1_appservice_cr.yaml
```

## run

### ansible

Runs as an ansible operator process. This is intended to be used when running
in a Pod inside a cluster. Developers wanting to run their operator locally
should use `up local` instead.

#### Flags

* `--reconcile-period` string - Default reconcile period for controllers (default 1m0s)
* `--watches-file` string - Path to the watches file to use (default "./watches.yaml")

#### Example

```console
$ operator-sdk run ansible --watches-file=/opt/ansible/watches.yaml --reconcile-period=30s
```

### helm

Runs as a helm operator process. This is intended to be used when running
in a Pod inside a cluster. Developers wanting to run their operator locally
should use `up local` instead.

#### Flags

* `--reconcile-period` string - Default reconcile period for controllers (default 1m0s)
* `--watches-file` string - Path to the watches file to use (default "./watches.yaml")

#### Example

```console
$ operator-sdk run helm --watches-file=/opt/helm/watches.yaml --reconcile-period=30s
```

## scorecard

Run scorecard tests on an operator

### Flags

* `basic-tests` - Enable basic operator checks (default true)
* `cr-manifest` string - (required) Path to manifest for Custom Resource
* `csv-path` string - (required if `olm-tests` is set) Path to CSV being tested
* `global-manifest` string - Path to manifest for Global resources (e.g. CRD manifests)
* `init-timeout` int - Timeout for status block on CR to be created, in seconds (default 10)
* `kubeconfig` string - Path to kubeconfig of custom resource created in cluster
* `namespace` string - Namespace of custom resource created in cluster
* `namespaced-manifest` string - Path to manifest for namespaced resources (e.g. RBAC and Operator manifest)
* `olm-deployed` - Only use the CSV at `csv-path` for manifest data, except for those provided to `cr-manifest`
* `olm-tests` - Enable OLM integration checks (default true)
* `proxy-image` string - Image name for scorecard proxy (default "quay.io/operator-framework/scorecard-proxy")
* `proxy-pull-policy` string - Pull policy for scorecard proxy image (default "Always")
* `verbose` - Enable verbose logging
* `-h, --help` - help for scorecard

### Example

```console
$ operator-sdk scorecard --cr-manifest deploy/crds/cache_v1alpha1_memcached_cr.yaml --csv-path deploy/olm-catalog/memcached-operator/0.0.2/memcached-operator.v0.0.2.clusterserviceversion.yaml
Checking for existence of spec and status blocks in CR
Checking that operator actions are reflected in status
Checking that writing into CRs has an effect
Checking for CRD resources
Checking for existence CR example
Checking spec descriptors
Checking status descriptors
Basic Operator:
        Spec Block Exists: 1/1 points
        Status Block Exist: 1/1 points
        Operator actions are reflected in status: 1/1 points
        Writing into CRs has an effect: 1/1 points
OLM Integration:
        Owned CRDs have resources listed: 1/1 points
        CRs have at least 1 example: 0/1 points
        Spec fields with descriptors: 1/1 points
        Status fields with descriptors: 0/1 points

Total Score: 6/8 points
SUGGESTION: Add an alm-examples annotation to your CSV to pass the CRs have at least 1 example test
SUGGESTION: Add a status descriptor for nodes
```

## test

### Available Commands

#### local

Runs the tests locally

##### Args

* `test-location` - location of e2e test files (e.g. "./test/e2e/")

##### Flags

* `--debug` - Enable debug-level logging
* `--kubeconfig` string - location of kubeconfig for Kubernetes cluster (default "~/.kube/config")
* `--global-manifest` string - path to manifest for global resources (default "deploy/crd.yaml)
* `--namespaced-manifest` string - path to manifest for per-test, namespaced resources (default: combines deploy/service_account.yaml, deploy/rbac.yaml, and deploy/operator.yaml)
* `--namespace` string - if non-empty, single namespace to run tests in (e.g. "operator-test") (default: "")
* `--go-test-flags` string - Additional flags to pass to go test
* `--molecule-test-flags` string - Additional flags to pass to molecule test
* `--up-local` - enable running operator locally with go run instead of as an image in the cluster
* `--no-setup` - disable test resource creation
* `--image` string - use a different operator image from the one specified in the namespaced manifest
* `-h, --help` - help for local

##### Use

The operator-sdk test command runs go tests built using the Operator SDK's test framework.

##### Example

```console
$ operator-sdk test local ./test/e2e/
ok    github.com/operator-framework/operator-sdk-samples/memcached-operator/test/e2e  20.410s
```

#### cluster

Runs the e2e tests packaged in an operator image as a pod in the cluster

##### Args

* `image-name` - the operator image that is used to run the tests in a pod (e.g. "quay.io/example/memcached-operator:v0.0.1")

##### Flags

* `--kubeconfig` string - location of kubeconfig for Kubernetes cluster (default "~/.kube/config")
* `--image-pull-policy` string - set test pod image pull policy. Allowed values: Always, Never (default "Always")
* `--namespace` string - namespace to run tests in (default "default")
* `--pending-timeout` int - timeout in seconds for testing pod to stay in pending state (default 60s)
* `--service-account` string - service account to run tests on (default "default")
* `-h, --help` - help for cluster

##### Use

The operator-sdk test command runs go tests embedded in an operator image built using the Operator SDK.

##### Example

```console
$ operator-sdk test cluster quay.io/example/memcached-operator:v0.0.1
Test Successfully Completed
```

## up

### Available Commands

#### local - Launches the operator locally

##### Use

The `operator-sdk up local` command launches the operator on the local machine
with the ability to access a Kubernetes cluster using a kubeconfig file, and
setting any necessary environment variables that the operator would expect to
find when running in a cluster. For Go-based operators, this command will
compile and run the operator binary. In the case of non-Go operators, it runs
the operator-sdk binary itself as the operator.

##### Flags

* `--go-ldflags` string - Set Go linker options
* `--kubeconfig` string - The file path to Kubernetes configuration file; defaults to $HOME/.kube/config
* `--namespace` string - The namespace where the operator watches for changes. (default "default")
* `--operator-flags` string - Flags that the local operator may need.
* `-h, --help` - help for local

##### Example

```console
$ operator-sdk up local --kubeconfig "mycluster.kubecfg" --namespace "default" --operator-flags "--flag1 value1 --flag2=value2"
```

The below example will use the default kubeconfig, the default namespace environment var, and pass in flags for the operator.
To use the operator flags, your operator must know how to handle the option. Below imagine an operator that understands the `resync-interval` flag.

```console
$ operator-sdk up local --operator-flags "--resync-interval 10"
```

If you are planning on using a different namespace than the default, then you should use the `--namespace` flag to change where the operator is watching for custom resources to be created.
For this to work your operator must handle the `WATCH_NAMESPACE` environment variable. To do that you can use the [utility function][utility_link] `k8sutil.GetWatchNamespace` in your operator.

```console
$ operator-sdk up local --namespace "testing"
```

### Flags

* `-h, --help` - help for up

[utility_link]: https://github.com/operator-framework/operator-sdk/blob/89bf021063d18b6769bdc551ed08fc37027939d5/pkg/util/k8sutil/k8sutil.go#L140
[k8s-code-generator]: https://github.com/kubernetes/code-generator
[openapi-code-generator]: https://github.com/kubernetes/kube-openapi
[helm-user-guide-create-project]: https://github.com/operator-framework/operator-sdk/blob/master/doc/helm/user-guide.md#create-a-new-project
