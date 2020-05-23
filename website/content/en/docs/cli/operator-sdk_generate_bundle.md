---
title: "operator-sdk generate bundle"
---
## operator-sdk generate bundle

Generates bundle data for the operator

### Synopsis

Generates bundle data for the operator

```
operator-sdk generate bundle [flags]
```

### Options

```
      --apis-dir string          Root directory for API type defintions
      --channels string          A comma-separated list of channels the bundle belongs to (default "alpha")
      --crds-dir string          Root directory for CustomResoureDefinition manifests
      --default-channel string   The default channel for the bundle
  -h, --help                     help for bundle
      --input-dir string         Directory to read an existing bundle from. This directory is the parent of your bundle 'manifests' directory, and different from --manifest-root
      --interactive              When set or no bundle base exists, an interactive command prompt will be presented to accept bundle ClusterServiceVersion metadata
      --manifests                Generate bundle manifests
      --metadata                 Generate bundle metadata and Dockerfile
      --operator-name string     Name of the bundle's operator
      --output-dir string        Directory to write the bundle to
      --overwrite                Overwrite the bundle's metadata and Dockerfile if they exist
  -q, --quiet                    Run in quiet mode
  -v, --version string           Semantic version of the operator in the generated bundle. Only set if creating a new bundle or upgrading your operator
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

