## operator-sdk olm-catalog gen-csv

Generates a Cluster Service Version yaml file for the operator

### Synopsis

The gen-csv command generates a Cluster Service Version (CSV) YAML manifest
for the operator. This file is used to publish the operator to the OLM Catalog.

A CSV semantic version is supplied via the --csv-version flag. If your operator
has already generated a CSV manifest you want to use as a base, supply its
version to --from-version. Otherwise the SDK will scaffold a new CSV manifest.

Configure CSV generation by writing a config file 'deploy/olm-catalog/csv-config.yaml

```
operator-sdk olm-catalog gen-csv [flags]
```

### Options

```
      --csv-channel string     Channel the CSV should be registered under in the package manifest
      --csv-config string      Path to CSV config file. Defaults to deploy/olm-catalog/csv-config.yaml
      --csv-version string     Semantic version of the CSV
      --default-channel        Use the channel passed to --csv-channel as the package manifests' default channel. Only valid when --csv-channel is set
      --from-version string    Semantic version of an existing CSV to use as a base
  -h, --help                   help for gen-csv
      --operator-name string   Operator name to use while generating CSV
      --update-crds            Update CRD manifests in deploy/{operator-name}/{csv-version} the using latest API's
```

### SEE ALSO

* [operator-sdk olm-catalog](operator-sdk_olm-catalog.md)	 - Invokes a olm-catalog command

