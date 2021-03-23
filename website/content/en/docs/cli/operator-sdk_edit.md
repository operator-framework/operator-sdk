---
title: "operator-sdk edit"
---
## operator-sdk edit

This command will edit the project configuration

### Synopsis

This command will edit the project configuration. You can have single or multi group project.

```
operator-sdk edit [flags]
```

### Examples

```
# Enable the multigroup layout
        operator-sdk edit --multigroup

        # Disable the multigroup layout
        operator-sdk edit --multigroup=false

```

### Options

```
  -h, --help         help for edit
      --multigroup   enable or disable multigroup layout
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

