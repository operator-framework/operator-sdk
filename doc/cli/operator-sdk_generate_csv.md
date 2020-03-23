## operator-sdk generate csv

Generates a ClusterServiceVersion YAML file for the operator

### Synopsis

The 'generate csv' command generates a ClusterServiceVersion (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

CSV input flags:
	--deploy-dir:	The CSV file contents will be generated from the operator manifests
					present in this directory.

	--apis-dir:		The CSV annotation comments will be parsed from the Go types under this path to 
					fill out metadata for owned APIs in spec.customresourcedefinitions.owned.

	--crd-dir:		The CRD manifests are updated from this path to the CSV bundle directory.
					Note: The CSV generator only uses this to copy the CRD manifests.
					The CSV contents for spec.customresourcedefinitions.owned will still be generated
					from the CRD manifests in the deploy directory specified by --deploy-dir.
					If unset, it defaults to the same value as --deploy-dir.



```
operator-sdk generate csv [flags]
```

### Examples

```

		##### Generate CSV from default input paths #####
		$ tree pkg/apis/ deploy/
		pkg/apis/
		├── ...
		└── cache
			├── group.go
			└── v1alpha1
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

		$ operator-sdk generate csv --csv-version=0.0.1 --update-crds
		INFO[0000] Generating CSV manifest version 0.0.1
		...

		$ tree deploy/
		deploy/
		...
		├── olm-catalog
		│   └── memcached-operator
		│       ├── 0.0.1
		│       │   ├── cache.example.com_memcacheds_crd.yaml
		│       │   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
		│       └── memcached-operator.package.yaml
		...



		##### Generate CSV from custom input paths #####
		$ operator-sdk generate csv --csv-version=0.0.1 --update-crds \
		--deploy-dir=config --apis-dir=api --output-dir=production
		INFO[0000] Generating CSV manifest version 0.0.1
		...

		$ tree config/ api/ production/
		config/
		├── crds
		│   ├── cache.example.com_memcacheds_crd.yaml
		│   └── cache.example.com_v1alpha1_memcached_cr.yaml
		├── operator.yaml
		├── role.yaml
		├── role_binding.yaml
		└── service_account.yaml
		api/
		├── ...
		└── cache
			├── group.go
			└── v1alpha1
				├── ...
				└── memcached_types.go
		production/
		└── olm-catalog
			└── memcached-operator
				├── 0.0.1
				│   ├── cache.example.com_memcacheds_crd.yaml
				│   └── memcached-operator.v0.0.1.clusterserviceversion.yaml
				└── memcached-operator.package.yaml

```

### Options

```
      --apis-dir string        Project relative path to root directory for API type defintions (default "pkg/apis")
      --crd-dir string         Project relative path to root directory for for CRD manifests
      --csv-channel string     Channel the CSV should be registered under in the package manifest
      --csv-version string     Semantic version of the CSV
      --default-channel        Use the channel passed to --csv-channel as the package manifests' default channel. Only valid when --csv-channel is set
      --deploy-dir string      Project relative path to root directory for operator manifests (Deployment, RBAC, CRDs) (default "deploy")
      --from-version string    Semantic version of an existing CSV to use as a base
  -h, --help                   help for csv
      --operator-name string   Operator name to use while generating CSV
      --output-dir string      Base directory to output generated CSV. The resulting CSV bundle directorywill be "<output-dir>/olm-catalog/<operator-name>/<csv-version>" (default "deploy")
      --update-crds            Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's
```

### SEE ALSO

* [operator-sdk generate](operator-sdk_generate.md)	 - Invokes a specific generator

