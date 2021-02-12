---
title: "operator-sdk generate bundle"
---
## operator-sdk generate bundle

Generates bundle data for the operator

### Synopsis


Running 'generate bundle' is the first step to publishing your operator to a catalog and/or deploying it with OLM.
This command generates a set of bundle manifests, metadata, and a bundle.Dockerfile for your operator.
A ClusterServiceVersion manifest will be generated from the set of manifests passed to this command (see below)
using either an existing base at '&lt;kustomize-dir&gt;/bases/&lt;package-name&gt;.clusterserviceversion.yaml',
typically containing metadata added by 'generate kustomize manifests' or by hand, or from scratch if that base
does not exist. All non-metadata values in a base will be overwritten.

There are two ways to pass cluster-ready manifests to this command: stdin via a Unix pipe,
or in a directory using '--input-dir'. See command help for more information on these modes.
Passing a directory is useful for running this command outside of a project, as kustomize
config files are likely not present and/or only cluster-ready manifests are available.

Set '--version' to supply a semantic version for your bundle if you are creating one
for the first time or upgrading an existing one.

If '--output-dir' is set and you wish to build bundle images from that directory,
either manually update your bundle.Dockerfile or set '--overwrite'.

More information on bundles:
https://github.com/operator-framework/operator-registry/#manifest-format


```
operator-sdk generate bundle [flags]
```

### Examples

```

  # If running within a project or in a project that uses kustomize to generate manifests,
	# make sure a kustomize directory exists that looks like the following 'config/manifests' directory:
  $ tree config/manifests
  config/manifests
  ├── bases
  │   └── memcached-operator.clusterserviceversion.yaml
  └── kustomization.yaml

  # Generate a 0.0.1 bundle by passing manifests to stdin:
  $ kustomize build config/manifests | operator-sdk generate bundle --version 0.0.1
  Generating bundle version 0.0.1
  ...

  # If running outside of a project or in a project that does not use kustomize to generate manifests,
	# make sure cluster-ready manifests are available on disk:
  $ tree deploy/
  deploy/
  ├── crds
  │   └── cache.my.domain_memcacheds.yaml
  ├── deployment.yaml
  ├── role.yaml
  ├── role_binding.yaml
  ├── service_account.yaml
  └── webhooks.yaml

  # Generate a 0.0.1 bundle by passing manifests by dir:
  $ operator-sdk generate bundle --input-dir deploy --version 0.0.1
  Generating bundle version 0.0.1
  ...

  # After running in either of the above modes, you should see this directory structure:
  $ tree bundle/
  bundle/
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

```

### Options

```
      --channels string          A comma-separated list of channels the bundle belongs to (default "alpha")
      --crds-dir string          Directory to read cluster-ready CustomResoureDefinition manifests from. This option can only be used if --deploy-dir is set
      --default-channel string   The default channel for the bundle
      --deploy-dir string        Directory to read cluster-ready operator manifests from. If --crds-dir is not set, CRDs are ready from this directory. This option is mutually exclusive with --input-dir and piping to stdin
  -h, --help                     help for bundle
      --input-dir string         Directory to read cluster-ready operator manifests from. This option is mutually exclusive with --deploy-dir/--crds-dir and piping to stdin. This option should not be passed an existing bundle directory, as this bundle will not contain the correct set of manifests required to generate a CSV. Use --kustomize-dir to pass a base CSV
      --kustomize-dir string     Directory containing kustomize bases in a "bases" dir and a kustomization.yaml for operator-framework manifests (default "config/manifests")
      --manifests                Generate bundle manifests
      --metadata                 Generate bundle metadata and Dockerfile
      --output-dir string        Directory to write the bundle to
      --overwrite                Overwrite the bundle's metadata and Dockerfile if they exist (default true)
      --package string           Bundle's package name
  -q, --quiet                    Run in quiet mode
      --stdout                   Write bundle manifest to stdout
  -v, --version string           Semantic version of the operator in the generated bundle. Only set if creating a new bundle or upgrading your operator
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

