---
title: "operator-sdk alpha config-3alpha-to-3"
---
## operator-sdk alpha config-3alpha-to-3

Convert your PROJECT config file from version 3-alpha to 3

### Synopsis

Your PROJECT file contains config data specified by some version.
This version is not a kubernetes-style version. In general, alpha and beta config versions
are unstable and support for them is dropped once a stable version is released.
The 3-alpha version has recently become stable (3), and therefore is no longer
supported by operator-sdk v1.5+. This command is intended to migrate 3-alpha PROJECT files
to 3 with as few manual modifications required as possible.


```
operator-sdk alpha config-3alpha-to-3 [flags]
```

### Options

```
  -h, --help   help for config-3alpha-to-3
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands

