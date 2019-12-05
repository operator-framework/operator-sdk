## operator-sdk scorecard

Run scorecard tests

### Synopsis

Runs blackbox scorecard tests on an operator


```
operator-sdk scorecard [flags]
```

### Options

```
  -b, --bundle string       OLM bundle directory path, when specified runs bundle validation
      --config string       config file (default is '<project_dir>/.osdk-scorecard'; the config file's extension and format can be .yaml, .json, or .toml)
  -h, --help                help for scorecard
      --kubeconfig string   Path to kubeconfig of custom resource created in cluster
  -L, --list                If true, only print the test names that would be run based on selector filtering (only valid when version is v1alpha2)
  -o, --output string       Output format for results. Valid values: text, json (default "text")
  -l, --selector string     selector (label query) to filter tests on (only valid when version is v1alpha2)
      --version string      scorecard version. Valid values: v1alpha1, v1alpha2 (default "v1alpha2")
```

### SEE ALSO

* [operator-sdk](operator-sdk.md)	 - An SDK for building operators with ease

