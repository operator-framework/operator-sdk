---
title: "operator-sdk"
---
## operator-sdk

Development kit for building Kubernetes extensions and tools.

### Synopsis

Development kit for building Kubernetes extensions and tools.

Provides libraries and tools to create new projects, APIs and controllers.
Includes tools for packaging artifacts into an installer container.

Typical project lifecycle:

- initialize a project:

  operator-sdk init --domain example.com --license apache2 --owner "The Kubernetes authors"

- create one or more a new resource APIs and add your code to them:

  operator-sdk create api --group <group> --version <version> --kind <Kind>

Create resource will prompt the user for if it should scaffold the Resource and / or Controller. To only
scaffold a Controller for an existing Resource, select "n" for Resource. To only define
the schema for a Resource without writing a Controller, select "n" for Controller.

After the scaffold is written, api will run make on the project.


```
operator-sdk [flags]
```

### Examples

```

  # Initialize your project
  operator-sdk init --domain example.com --license apache2 --owner "The Kubernetes authors"

  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  operator-sdk create api --group ship --version v1beta1 --kind Frigate

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano controllers/frigate_controller.go

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run

```

### Options

```
  -h, --help      help for operator-sdk
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk bundle](../operator-sdk_bundle)	 - Manage operator bundle metadata
* [operator-sdk cleanup](../operator-sdk_cleanup)	 - Clean up an Operator deployed with the 'run' subcommand
* [operator-sdk completion](../operator-sdk_completion)	 - Generators for shell completions
* [operator-sdk create](../operator-sdk_create)	 - Scaffold a Kubernetes API or webhook
* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator
* [operator-sdk init](../operator-sdk_init)	 - Initialize a new project
* [operator-sdk olm](../operator-sdk_olm)	 - Manage the Operator Lifecycle Manager installation in your cluster
* [operator-sdk run](../operator-sdk_run)	 - Run an Operator in a variety of environments
* [operator-sdk scorecard](../operator-sdk_scorecard)	 - Runs scorecard
* [operator-sdk version](../operator-sdk_version)	 - Prints the version of operator-sdk

