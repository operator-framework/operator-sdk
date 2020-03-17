## operator-sdk new

Creates a new operator application

### Synopsis

The operator-sdk new command creates a new operator application and
generates a default directory layout based on the input <project-name>.

<project-name> is the project name of the new operator. (e.g app-operator)

For example:

	$ mkdir $HOME/projects/example.com/
	$ cd $HOME/projects/example.com/
	$ operator-sdk new app-operator
generates a skeletal app-operator application in $HOME/projects/example.com/app-operator.


```
operator-sdk new <project-name> [flags]
```

### Options

```
      --api-version string          Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1) - used with "ansible" or "helm" types
      --generate-playbook           Generate a playbook skeleton. (Only used for --type ansible)
      --git-init                    Initialize the project directory as a git repository (default false)
      --header-file string          Path to file containing headers for generated Go files. Copied to hack/boilerplate.go.txt
      --helm-chart string           Initialize helm operator with existing helm chart (<URL>, <repo>/<name>, or local path)
      --helm-chart-repo string      Chart repository URL for the requested helm chart
      --helm-chart-version string   Specific version of the helm chart (default is latest version)
  -h, --help                        help for new
      --kind string                 Kubernetes CustomResourceDefintion kind. (e.g AppService) - used with "ansible" or "helm" types
      --repo string                 Project repository path for Go operators. Used as the project's Go import path. This must be set if outside of $GOPATH/src (e.g. github.com/example-inc/my-operator)
      --skip-validation             Do not validate the resulting project's structure and dependencies. (Only used for --type go)
      --type string                 Type of operator to initialize (choices: "go", "ansible" or "helm") (default "go")
      --vendor                      Use a vendor directory for dependencies
```

### SEE ALSO

* [operator-sdk](operator-sdk.md)	 - An SDK for building operators with ease

