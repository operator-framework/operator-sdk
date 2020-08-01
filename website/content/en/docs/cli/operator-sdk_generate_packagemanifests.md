---
title: "operator-sdk generate packagemanifests"
---
## operator-sdk generate packagemanifests

Generates package manifests data for the operator

### Synopsis


Note: while the package manifests format is not yet deprecated, the operator-framework is migrated
towards using bundles by default. Run 'operator-sdk generate bundle -h' for more information.

Running 'generate packagemanifests' is the first step to publishing your operator to a catalog and/or deploying
it with OLM. This command generates a set of manifests in a versioned directory and a package manifest file for
your operator. Typically one would run 'generate kustomize manifests' first to (re)generate kustomize bases
consumed by this command.

Set '--version' to supply a semantic version for your new package. This is a required flag when running
'generate packagemanifests --manifests'.

More information on the package manifests format:
https://github.com/operator-framework/operator-registry/#manifest-format


```
operator-sdk generate packagemanifests [flags]
```

### Examples

```

  # Generate manifests then create the package manifests base:
  $ make manifests
  /home/user/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
  $ operator-sdk generate kustomize manifests

  Display name for the operator (required):
  > memcached-operator
  ...

  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml
  $ kustomize build config/manifests | operator-sdk generate packagemanifests --manifests --version 0.0.1
  Generating package manifests version 0.0.1
  ...

  # After running the above commands, you should see this directory structure:
  $ tree packagemanifests
  packagemanifests
  ├── 0.0.1
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── memcached-operator.package.yaml

```

### Options

```
      --channel string         Channel name for the generated package
      --crds-dir string        Root directory for CustomResoureDefinition manifests
      --default-channel        Use the channel passed to --channel as the package manifest file's default channel
      --deploy-dir string      Root directory for operator manifests such as Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir
      --from-version string    Semantic version of the operator being upgraded from
  -h, --help                   help for packagemanifests
      --input-dir string       Directory to read existing package manifests from. This directory is the parent of individual versioned package directories, and different from --deploy-dir
      --kustomize-dir string   Directory containing kustomize bases and a kustomization.yaml for operator-framework manifests (default "config/manifests")
      --output-dir string      Directory in which to write package manifests
  -q, --quiet                  Run in quiet mode
      --stdout                 Write package to stdout
      --update-objects         Update non-CSV objects in this package, ex. CustomResoureDefinitions, Roles (default true)
  -v, --version string         Semantic version of the packaged operator
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

