---
title: "operator-sdk generate packagemanifests"
---
## operator-sdk generate packagemanifests

Generates a package manifests format

### Synopsis


Note: while the package manifests format is not yet deprecated, the operator-framework is migrated
towards using bundles by default. Run 'operator-sdk generate bundle -h' for more information.

Running 'generate packagemanifests' is the first step to publishing your operator to a catalog
and/or deploying it with OLM. This command generates a set of manifests in a versioned directory
and a package manifest file for your operator. It will interactively ask for UI metadata,
an important component of publishing your operator, by default unless a package for your
operator exists or you set '--interactive=false'.

Set '--version' to supply a semantic version for your new package. This is a required flag when running
'generate packagemanifests --manifests'.

More information on the package manifests format:
https://github.com/operator-framework/operator-registry/#manifest-format


```
operator-sdk generate packagemanifests [flags]
```

### Examples

```

  # Create the package manifest file and a new package:
  $ operator-sdk generate packagemanifests --version 0.0.1
  INFO[0000] Generating package manifests version 0.0.1

  Display name for the operator (required):
  > memcached-operator
  ...

  # After running the above commands, you should see:
  $ tree deploy/olm-catalog
  deploy/olm-catalog
  └── memcached-operator
      ├── 0.0.1
      │   ├── cache.example.com_memcacheds_crd.yaml
      │   └── memcached-operator.clusterserviceversion.yaml
      └── memacached-operator.package.yaml

```

### Options

```
      --apis-dir string        Root directory for API type defintions
      --channel string         Channel name for the generated package
      --crds-dir string        Root directory for CustomResoureDefinition manifests
      --default-channel        Use the channel passed to --channel as the package manifest file's default channel
      --deploy-dir string      Root directory for operator manifests such as Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir
      --from-version string    Semantic version of the operator being upgraded from
  -h, --help                   help for packagemanifests
      --input-dir string       Directory to read existing package manifests from. This directory is the parent of individual versioned package directories, and different from --deploy-dir
      --interactive            When set or no package base exists, an interactive command prompt will be presented to accept package ClusterServiceVersion metadata
      --operator-name string   Name of the packaged operator
      --output-dir string      Directory in which to write package manifests
  -q, --quiet                  Run in quiet mode
      --update-crds            Update CustomResoureDefinition manifests in this package (default true)
  -v, --version string         Semantic version of the packaged operator
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

