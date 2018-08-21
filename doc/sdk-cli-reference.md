# CLI Guide

```bash
Usage:
  operator-sdk [command]
```

## build

### Args

* image - is the container image to be built, e.g. "quay.io/example/operator:v0.0.1". This image will be automatically set in the deployment manifests.

### Flags

* `-h, --help` - help for build

### Use

The operator-sdk build command compiles the code, builds the executables,
and generates Kubernetes manifests. After build completes, the image would be built locally in docker. Then it needs to be pushed to remote registry.

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

## generate

### Available Commands

#### k8s - Generates Kubernetes code for custom resource

##### Use

k8s generator generates code for custom resource given the API spec
to comply with kube-API requirements.

##### Flags

* `-h, --help` - help for k8s

##### Example

```bash
operator-sdk generate k8s

# Output:
Run code-generation for custom resources
Generating deepcopy funcs
```

#### olm-catalog - Generates OLM Catalog manifests

##### Flags

* `--image` **(required)** string - The container image name to set in the CSV to deploy the operator e.g: quay.io/example/operator:v0.0.1
* `--version` **(required)** string - The version of the current CSV e.g: 0.0.1
* `-h, --help` - help for olm-catalog

##### Example

```bash
operator-sdk generate olm-catalog --image=quay.io/example/operator:v0.0.1 --version=0.0.1

# Output:
Generating OLM catalog manifests
```

### Flags

* `-h, --help` - help for generate

## new

The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input `project-name`.

### Args

* `project-name` - the project name of the new

### Flags

* `--api-version` **(required)** string - Kubernetes apiVersion and has a format of `$GROUP_NAME/$VERSION` (e.g app.example.com/v1alpha1)
* `--kind` **(required)** string - Kubernetes CustomResourceDefintion kind. (e.g AppService)
* `-h, --help` - help for new

### Example

```bash
mkdir $GOPATH/src/github.com/example.com/
cd $GOPATH/src/github.com/example.com/
operator-sdk new app-operator --api-version=app.example.com/v1alpha1 --kind=AppService

# Output:
Create app-operator/.gitignore
...
```

## test

### Flags

* `-t, --test-location` **(required)** string - location of e2e test files
* `-k, --kubeconfig` string - location of kubeconfig for kubernetes cluster
* `-g, --global-init` string - location of global resource manifest yaml file
* `-n, --namespaced-init` string - location of namespaced resource manifest yaml file
* `-f, --go-test-flags` string - extra arguments to pass to `go test` (e.g. -f "-v -parallel=2")
* `-h, --help` - help for test

### Use

The operator-sdk test command runs go tests built using the Operator SDK's test framework.

### Example:

#### Test

```bash
operator-sdk test --test-location ./test/e2e/

# Output:
ok  	github.com/operator-framework/operator-sdk-samples/memcached-operator/test/e2e	20.410s
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
