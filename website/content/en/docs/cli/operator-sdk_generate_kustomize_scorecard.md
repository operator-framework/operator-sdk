---
title: "operator-sdk generate kustomize scorecard"
---
## operator-sdk generate kustomize scorecard

Generates scorecard configuration files

### Synopsis


Running 'generate kustomize scorecard' will (re)generate scorecard configuration kustomize bases,
default test patches, and a kustomization.yaml in 'config/scorecard'.


```
operator-sdk generate kustomize scorecard [flags]
```

### Examples

```

  $ operator-sdk generate kustomize scorecard
  Generating kustomize files in config/scorecard
  Kustomize files generated successfully
  $ tree ./config/scorecard
  ./config/scorecard/
  ├── bases
  │   └── config.yaml
  ├── kustomization.yaml
  └── patches
      ├── basic.config.yaml
      └── olm.config.yaml

```

### Options

```
  -h, --help                    help for scorecard
      --image /scorecard-test   Image to use for default tests; this image must contain the /scorecard-test binary (default "quay.io/operator-framework/scorecard-test:latest")
      --operator-name string    Name of the operator
      --output-dir string       Directory to write kustomize files
  -q, --quiet                   Run in quiet mode
```

### Options inherited from parent commands

```
      --verbose   Enable verbose logging
```

### SEE ALSO

* [operator-sdk generate kustomize](../operator-sdk_generate_kustomize)	 - Contains subcommands that generate operator-framework kustomize data for the operator

