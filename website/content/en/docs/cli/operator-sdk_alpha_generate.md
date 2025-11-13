---
title: "operator-sdk alpha generate"
---
## operator-sdk alpha generate

Re-scaffold a Kubebuilder project from its PROJECT file

### Synopsis

The 'generate' command re-creates a Kubebuilder project scaffold based on the configuration 
defined in the PROJECT file, using the latest installed Kubebuilder version and plugins.

This is helpful for migrating projects to a newer Kubebuilder layout or plugin version (e.g., v3 to v4)
as update your project from any previous version to the current one.

If no output directory is provided, the current working directory will be cleaned (except .git and PROJECT).

```
operator-sdk alpha generate [flags]
```

### Examples

```

  # **WARNING**(will delete all files to allow the re-scaffold except .git and PROJECT)
  # Re-scaffold the project in-place 
  kubebuilder alpha generate

  # Re-scaffold the project from ./test into ./my-output
  kubebuilder alpha generate --input-dir="./path/to/project" --output-dir="./my-output"

```

### Options

```
  -h, --help                help for generate
      --input-dir string    Path to the directory containing the PROJECT file. Defaults to the current working directory. WARNING: delete existing files (except .git and PROJECT).
      --output-dir string   Directory where the new project scaffold will be written. If unset, re-scaffolding occurs in-place and will delete existing files (except .git and PROJECT).
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands

