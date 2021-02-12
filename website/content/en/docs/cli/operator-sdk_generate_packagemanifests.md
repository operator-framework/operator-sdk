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
your operator. A ClusterServiceVersion manifest will be generated from the set of manifests passed to this command
(see below) using either an existing base at '&lt;kustomize-dir&gt;/bases/&lt;package-name&gt;.clusterserviceversion.yaml',
typically containing metadata added by 'generate kustomize manifests' or by hand, or from scratch if that base
does not exist. All non-metadata values in a base will be overwritten.

There are two ways to pass cluster-ready manifests to this command: stdin via a Unix pipe,
or in a directory using '--input-dir'. See command help for more information on these modes.
Passing a directory is useful for running this command outside of a project, as kustomize
config files are likely not present and/or only cluster-ready manifests are available.

Set '--version' to supply a semantic version for your new package.

More information on the package manifests format:
https://github.com/operator-framework/operator-registry/#manifest-format


```
operator-sdk generate packagemanifests [flags]
```

### Examples

```

  # If running within a project, make sure a 'config' kustomize directory exists and has a 'config/manifests':
  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml

  # Generate a 0.0.1 packagemanifests by passing manifests to stdin:
  $ kustomize build config/manifests | operator-sdk generate packagemanifests --version 0.0.1
  Generating package manifests version 0.0.1
  ...

  # If running outside of a project, make sure cluster-ready manifests are available on disk:
  $ tree deploy/
  deploy/
  ├── crds
  │   └── cache.my.domain_memcacheds.yaml
  ├── deployment.yaml
  ├── role.yaml
  ├── role_binding.yaml
  ├── service_account.yaml
  └── webhooks.yaml

  # Generate a 0.0.1 packagemanifests by passing manifests by dir:
  $ operator-sdk generate packagemanifests --deploy-dir deploy --version 0.0.1
  Generating package manifests version 0.0.1
  ...

  # After running in either of the above modes, you should see this directory structure:
  $ tree packagemanifests/
  packagemanifests/
  ├── 0.0.1
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── memcached-operator.package.yaml

```

### Options

```
      --channel string         Channel name for the generated package
      --crds-dir string        Directory to read cluster-ready CustomResoureDefinition manifests from. This option can only be used if --deploy-dir is set
      --default-channel        Use the channel passed to --channel as the package manifest file's default channel
      --deploy-dir string      Directory to read cluster-ready operator manifests from. If --crds-dir is not set, CRDs are ready from this directory
      --from-version string    Semantic version of the operator being upgraded from
  -h, --help                   help for packagemanifests
      --input-dir string       Directory to read existing package manifests from. This directory is the parent of individual versioned package directories, and different from --deploy-dir (default "packagemanifests")
      --kustomize-dir string   Directory containing kustomize bases in a "bases" dir and a kustomization.yaml for operator-framework manifests (default "config/manifests")
      --output-dir string      Directory in which to write package manifests
      --package string         Package name
  -q, --quiet                  Run in quiet mode
      --stdout                 Write package to stdout
      --update-objects         Update non-CSV objects in this package, ex. CustomResoureDefinitions, Roles (default true)
  -v, --version string         Semantic version of the packaged operator
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

