---
title: "operator-sdk init"
---
## operator-sdk init

Initialize a new project

### Synopsis

Initialize a new project including vendor/ directory and Go package directories.

Writes the following files:
- a boilerplate license file
- a PROJECT file with the domain and repo
- a Makefile to build the project
- a go.mod with project dependencies
- a Kustomization.yaml for customizating manifests
- a Patch file for customizing image for manager manifests
- a Patch file for enabling prometheus metrics
- a cmd/manager/main.go to run

project will prompt the user to run 'dep ensure' after writing the project files.


```
operator-sdk init [flags]
```

### Examples

```
  # Scaffold a project using the apache2 license with "The Kubernetes authors" as owners
  operator-sdk init --project-version=2 --domain example.org --license apache2 --owner "The Kubernetes authors"

```

### Options

```
      --domain string            domain for groups (default "my.domain")
      --fetch-deps               ensure dependencies are downloaded (default true)
  -h, --help                     help for init
      --license string           license to use to boilerplate, may be one of 'apache2', 'none' (default "apache2")
      --owner string             owner to add to the copyright
      --plugins strings          Name and optionally version of the plugin to initialize the project with. Available plugins: ("ansible.sdk.operatorframework.io/v1", "go.kubebuilder.io/v2", "helm.sdk.operatorframework.io/v1")
      --project-name string      name of this project
      --project-version string   project version, possible values: ("2", "3-alpha") (default "3-alpha")
      --repo string              name to use for go module (e.g., github.com/user/repo), defaults to the go package of the current working directory.
      --skip-go-version-check    if specified, skip checking the Go version
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - Development kit for building Kubernetes extensions and tools.

