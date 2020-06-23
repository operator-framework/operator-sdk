---
title: "operator-sdk generate bundle"
---
## operator-sdk generate bundle

Generates bundle data for the operator

### Synopsis


Running 'generate bundle' is the first step to publishing your operator to a catalog
and/or deploying it with OLM. This command generates a set of bundle manifests,
metadata, and a bundle.Dockerfile for your operator, and will interactively ask
for UI metadata, an important component of publishing your operator, by default unless
a bundle for your operator exists or you set '--interactive=false'.

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

  # Create bundle manifests, metadata, and a bundle.Dockerfile:
  $ operator-sdk generate bundle --version 0.0.1
  INFO[0000] Generating bundle manifest version 0.0.1

  Display name for the operator (required):
  > memcached-operator
  ...

  # After running the above commands, you should see:
  $ tree deploy/olm-catalog
  deploy/olm-catalog
  └── memcached-operator
      ├── manifests
      │   ├── cache.example.com_memcacheds_crd.yaml
      │   └── memcached-operator.clusterserviceversion.yaml
      └── metadata
          └── annotations.yaml

  # Then build and push your bundle image:
  $ export USERNAME=<your registry username>
  $ export BUNDLE_IMG=quay.io/$USERNAME/memcached-operator-bundle:v0.0.1
  $ docker build -f bundle.Dockerfile -t $BUNDLE_IMG .
  Sending build context to Docker daemon  42.33MB
  Step 1/9 : FROM scratch
  ...
  $ docker push $BUNDLE_IMG

```

### Options

```
      --apis-dir string          Root directory for API type defintions
      --channels string          A comma-separated list of channels the bundle belongs to (default "alpha")
      --crds-dir string          Root directory for CustomResoureDefinition manifests
      --default-channel string   The default channel for the bundle
      --deploy-dir string        Root directory for operator manifests such as Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir
  -h, --help                     help for bundle
      --input-dir string         Directory to read an existing bundle from. This directory is the parent of your bundle 'manifests' directory, and different from --deploy-dir
      --interactive              When set or no bundle base exists, an interactive command prompt will be presented to accept bundle ClusterServiceVersion metadata
      --manifests                Generate bundle manifests
      --metadata                 Generate bundle metadata and Dockerfile
      --operator-name string     Name of the bundle's operator
      --output-dir string        Directory to write the bundle to
      --overwrite                Overwrite the bundle's metadata and Dockerfile if they exist (default true)
  -q, --quiet                    Run in quiet mode
  -v, --version string           Semantic version of the operator in the generated bundle. Only set if creating a new bundle or upgrading your operator
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

