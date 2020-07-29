---
title: "operator-sdk generate kustomize manifests"
---
## operator-sdk generate kustomize manifests

Generates kustomize bases and a kustomization.yaml for operator-framework manifests

### Synopsis


Running 'generate kustomize manifests' will (re)generate kustomize bases and a kustomization.yaml in
'config/manifests', which are used to build operator-framework manifests by other operator-sdk commands.
This command will interactively ask for UI metadata, an important component of manifest bases,
by default unless a base already exists or you set '--interactive=false'.


```
operator-sdk generate kustomize manifests [flags]
```

### Examples

```

  $ operator-sdk generate kustomize manifests

  Display name for the operator (required):
  > memcached-operator
  ...

  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml

  # After generating kustomize bases and a kustomization.yaml, you can generate a bundle or package manifests.

  # To generate a bundle:
  $ kustomize build config/manifests | operator-sdk generate bundle --version 0.0.1

  # To generate package manifests:
  $ kustomize build config/manifests | operator-sdk generate packagemanifests --version 0.0.1

```

### Options

```
      --apis-dir string     Root directory for API type defintions
  -h, --help                help for manifests
      --input-dir string    Directory containing existing kustomize files
      --interactive         When set or no kustomize base exists, an interactive command prompt will be presented to accept non-inferrable metadata
      --output-dir string   Directory to write kustomize files
  -q, --quiet               Run in quiet mode
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate kustomize](../operator-sdk_generate_kustomize)	 - Contains subcommands that generate operator-framework kustomize data for the operator

