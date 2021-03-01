---
title: "operator-sdk edit"
---
## operator-sdk edit

Update the project configuration

### Synopsis

This command will edit the project configuration.
Features supported:
  - Toggle between single or multi group projects.


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
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

