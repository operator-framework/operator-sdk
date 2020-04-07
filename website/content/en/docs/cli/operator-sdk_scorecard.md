---
title: "operator-sdk scorecard"
---
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
      --config string       config file (default is '<project_dir>/.osdk-scorecard.yaml'; the config file's extension and format must be .yaml
  -h, --help                help for scorecard
      --kubeconfig string   Path to kubeconfig of custom resource created in cluster
  -L, --list                If true, only print the test names that would be run based on selector filtering
  -o, --output string       Output format for results. Valid values: text, json (default "text")
  -l, --selector string     selector (label query) to filter tests on
      --version string      scorecard version. Valid values: v1alpha2 (default "v1alpha2")
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - An SDK for building operators with ease

