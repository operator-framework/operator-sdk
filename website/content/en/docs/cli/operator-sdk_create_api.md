---
title: "operator-sdk create api"
---
## operator-sdk create api

Scaffold a Kubernetes API

### Synopsis

Scaffold a Kubernetes API by writing a Resource definition and/or a Controller.

If information about whether the resource and controller should be scaffolded
was not explicitly provided, it will prompt the user if they should be.

After the scaffold is written, the dependencies will be updated and
make generate will be run.


```
operator-sdk create api [flags]
```

### Examples

```
  # Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
  operator-sdk create api --group ship --version v1beta1 --kind Frigate

  # Edit the API Scheme
  nano api/v1beta1/frigate_types.go

  # Edit the Controller
  nano controllers/frigate/frigate_controller.go

  # Edit the Controller Test
  nano controllers/frigate/frigate_controller_test.go

  # Install CRDs into the Kubernetes cluster using kubectl apply
  make install

  # Regenerate code and run against the Kubernetes cluster configured by ~/.kube/config
  make run

```

### Options

```
      --controller           if set, generate the controller without prompting the user (default true)
      --crd-version string   version of CustomResourceDefinition to scaffold. Options: [v1, v1beta1] (default "v1")
      --force                attempt to create resource even if it already exists
      --group string         resource Group
  -h, --help                 help for api
      --kind string          resource Kind
      --make make generate   if true, run make generate after generating files (default true)
      --namespaced           resource is namespaced (default true)
      --plural string        resource irregular plural form
      --resource             if set, generate the resource without prompting the user (default true)
      --version string       resource Version
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk create](../operator-sdk_create)	 - Scaffold a Kubernetes API or webhook

