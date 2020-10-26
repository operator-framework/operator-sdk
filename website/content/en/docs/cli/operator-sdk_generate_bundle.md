---
title: "operator-sdk generate bundle"
---
## operator-sdk generate bundle

Generates bundle data for the operator

### Synopsis


Running 'generate bundle' is the first step to publishing your operator to a catalog and/or deploying it with OLM.
This command generates a set of bundle manifests, metadata, and a bundle.Dockerfile for your operator.
Typically one would run 'generate kustomize manifests' first to (re)generate kustomize bases consumed by this command.

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

  # Generate bundle files and build your bundle image with these 'make' recipes:
  $ make bundle
  $ export USERNAME=<your registry username>
  $ export BUNDLE_IMG=quay.io/$USERNAME/memcached-operator-bundle:v0.0.1
  $ make bundle-build BUNDLE_IMG=$BUNDLE_IMG

  # The above recipe runs the following commands manually. First it creates bundle
  # manifests, metadata, and a bundle.Dockerfile:
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
  $ kustomize build config/manifests | operator-sdk generate bundle --overwrite --version 0.0.1
  Generating bundle manifest version 0.0.1
  ...

  # After running the above commands, you should see this directory structure:
  $ tree bundle
  bundle
  ├── manifests
  │   ├── cache.my.domain_memcacheds.yaml
  │   └── memcached-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml

  # Then it validates your bundle files and builds your bundle image:
  $ operator-sdk bundle validate ./bundle
  $ docker build -f bundle.Dockerfile -t $BUNDLE_IMG .
  Sending build context to Docker daemon  42.33MB
  Step 1/9 : FROM scratch
  ...

  # You can then push your bundle image:
  $ make docker-push IMG=$BUNDLE_IMG

```

### Options

```
      --channels string          A comma-separated list of channels the bundle belongs to (default "alpha")
      --crds-dir string          Root directory for CustomResoureDefinition manifests
      --default-channel string   The default channel for the bundle
      --deploy-dir string        Root directory for operator manifests such as Deployments and RBAC, ex. 'deploy'. This directory is different from that passed to --input-dir
  -h, --help                     help for bundle
      --input-dir string         Directory to read an existing bundle from. This directory is the parent of your bundle 'manifests' directory, and different from --deploy-dir
      --kustomize-dir string     Directory containing kustomize bases and a kustomization.yaml for operator-framework manifests (default "config/manifests")
      --manifests                Generate bundle manifests
      --metadata                 Generate bundle metadata and Dockerfile
      --output-dir string        Directory to write the bundle to
      --overwrite                Overwrite the bundle's metadata and Dockerfile if they exist (default true)
  -q, --quiet                    Run in quiet mode
      --stdout                   Write bundle manifest to stdout
  -v, --version string           Semantic version of the operator in the generated bundle. Only set if creating a new bundle or upgrading your operator
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

