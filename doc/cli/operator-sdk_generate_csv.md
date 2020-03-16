## operator-sdk generate csv

Generates a ClusterServiceVersion YAML file for the operator

### Synopsis

The 'generate csv' command generates a ClusterServiceVersion (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

```
operator-sdk generate csv [flags]
```

### Examples

```

TODO
	
```

### Options

```
      --apis-dir string        Project relative path to root directory for API type defintions.
                               The CSV annotation comments will be parsed from the Go types under this path to
                               fill out metadata for owned APIs in spec.customresourcedefinitions.owned.
                                (default "pkg/apis")
      --crd-dir string         Project relative path to root directory for for CRD manifests.
                               Used when --update-crds is set to copy over CRD manifests to the CSV bundle directory.
                               Note: The CSV generator only uses this to copy the CRD manifests.
                               The CSV contents for spec.customresourcedefinitions.owned will still be updated
                               from the CRD manifests in the deploy directory specified by --deploy-dir.
                               If unset, it defaults to the same value as --deploy-dir.
                               
      --csv-channel string     Channel the CSV should be registered under in the package manifest
      --csv-version string     Semantic version of the CSV
      --default-channel        Use the channel passed to --csv-channel as the package manifests' default channel. Only valid when --csv-channel is set
      --deploy-dir string      Project relative path to root directory for operator manifests (Deployment, RBAC, CRDs).
                               The CSV file contents will be generated from the manifests present in this directory. 
                                (default "deploy")
      --from-version string    Semantic version of an existing CSV to use as a base
  -h, --help                   help for csv
      --operator-name string   Operator name to use while generating CSV
      --output-dir string      Base directory to output generated CSV. The resulting CSV bundle directorywill be "<output-dir>/olm-catalog/<operator-name>/<csv-version>" (default "deploy")
      --update-crds            Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's
```

### SEE ALSO

* [operator-sdk generate](operator-sdk_generate.md)	 - Invokes a specific generator

