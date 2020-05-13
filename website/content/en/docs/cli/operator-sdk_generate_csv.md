---
title: "operator-sdk generate csv"
---
## operator-sdk generate csv

Generates a ClusterServiceVersion YAML file for the operator

### Synopsis

The 'generate csv' command generates a ClusterServiceVersion (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

The --make-manifests flag directs the generator to create a bundle manifests directory
intended to hold your latest operator manifests. This flag is true by default.

More information on bundles:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md#operator-bundle-overview

Flags that change project default paths:
  --deploy-dir:
    The CSV's install strategy and permissions will be generated from the operator manifests
    (Deployment and Role/ClusterRole) present in this directory.

  --apis-dir:
    The CSV annotation comments will be parsed from the Go types under this path to
    fill out metadata for owned APIs in spec.customresourcedefinitions.owned.

  --crd-dir:
    The CSV's spec.customresourcedefinitions.owned field is generated from the CRD manifests
    in this path. These CRD manifests are also copied over to the bundle directory if
    --update-crds=true (the default). Additionally the CR manifests will be used to populate
    the CSV example CRs.


```
operator-sdk generate csv [flags]
```

### Examples

```
    ##### Generate a CSV in bundle format from default input paths #####
    $ tree pkg/apis/ deploy/
    pkg/apis/
    ├── ...
    └── cache
        ├── group.go
        ├── v1alpha1
        ├── ...
        └── memcached_types.go
    deploy/
    ├── crds
    │   ├── cache.example.com_memcacheds_crd.yaml
    │   └── cache.example.com_v1alpha1_memcached_cr.yaml
    ├── operator.yaml
    ├── role.yaml
    ├── role_binding.yaml
    └── service_account.yaml

    $ operator-sdk generate csv --csv-version=0.0.1
    INFO[0000] Generating CSV manifest version 0.0.1
    ...

    $ tree deploy/
    deploy/
    ...
    └── olm-catalog
        └── memcached-operator
            └── manifests
                ├── cache.example.com_memcacheds_crd.yaml
                └── memcached-operator.clusterserviceversion.yaml
    ...

    ##### Generate a CSV in package manifests format from default input paths #####

		$ operator-sdk generate csv --csv-version=0.0.1 --make-manifests=false --update-crds
    INFO[0000] Generating CSV manifest version 0.0.1
    ...
    $ tree deploy/
    deploy/
    ...
    └── olm-catalog
        └── memcached-operator
            ├── 0.0.1
            │   ├── cache.example.com_memcacheds_crd.yaml
            │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
            └── memcached-operator.package.yaml
    ...

    ##### Generate CSV from custom input paths #####
    $ operator-sdk generate csv --csv-version=0.0.1 --update-crds \
    --deploy-dir=config --apis-dir=api --output-dir=production
    INFO[0000] Generating CSV manifest version 0.0.1
    ...

    $ tree config/ api/ production/
    config/
    ├── crds
    │   ├── cache.example.com_memcacheds_crd.yaml
    │   └── cache.example.com_v1alpha1_memcached_cr.yaml
    ├── operator.yaml
    ├── role.yaml
    ├── role_binding.yaml
    └── service_account.yaml
    api/
    ├── ...
    └── cache
    |   ├── group.go
    |   └── v1alpha1
    |       ├── ...
    |       └── memcached_types.go
    production/
    └── olm-catalog
        └── memcached-operator
            ├── 0.0.1
            │   ├── cache.example.com_memcacheds_crd.yaml
            │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
            └── memcached-operator.package.yaml

```

### Options

```
      --apis-dir string        Project relative path to root directory for API type defintions (default "pkg/apis")
      --crd-dir string         Project relative path to root directory for CRD and CR manifests
      --csv-channel string     Channel the CSV should be registered under in the package manifest
      --csv-version string     Semantic version of the CSV. This flag must be set if a package manifest exists
      --default-channel        Use the channel passed to --csv-channel as the package manifests' default channel. Only valid when --csv-channel is set
      --deploy-dir string      Project relative path to root directory for operator manifests (Deployment and RBAC) (default "deploy")
      --from-version string    Semantic version of an existing CSV to use as a base
  -h, --help                   help for csv
      --interactive            When set, will enable the interactive command prompt feature to fill the UI metadata fields in CSV
      --make-manifests         When set, the generator will create or update a CSV manifest in a 'manifests' directory. This directory is intended to be used for your latest bundle manifests. The default location is deploy/olm-catalog/<operator-name>/manifests. If --output-dir is set, the directory will be <output-dir>/manifests (default true)
      --operator-name string   Operator name to use while generating CSV
      --output-dir string      Base directory to output generated CSV. If --make-manifests=false the resulting CSV bundle directory will be <output-dir>/olm-catalog/<operator-name>/<csv-version>. If --make-manifests=true, the bundle directory will be <output-dir>/manifests
      --update-crds            Update CRD manifests in deploy/<operator-name>/<csv-version> from the default CRDs dir deploy/crds or --crd-dir if set. If --make-manifests=false, this option is false by default (default true)
```

### SEE ALSO

* [operator-sdk generate](../operator-sdk_generate)	 - Invokes a specific generator

