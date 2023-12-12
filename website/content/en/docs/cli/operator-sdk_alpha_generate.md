---
title: "operator-sdk alpha generate"
---
## operator-sdk alpha generate

Re-scaffold an existing Kuberbuilder project

### Synopsis

It's an experimental feature that has the purpose of re-scaffolding the whole project from the scratch 
using the current version of KubeBuilder binary available.
# make sure the PROJECT file is in the 'input-dir' argument, the default is the current directory.
$ kubebuilder alpha generate --input-dir="./test" --output-dir="./my-output"
Then we will re-scaffold the project by Kubebuilder in the directory specified by 'output-dir'.
		

```
operator-sdk alpha generate [flags]
```

### Options

```
  -h, --help                help for generate
      --input-dir string    path to a Kubebuilder project file if not in the current working directory
      --output-dir string   path to output the scaffolding. defaults a directory in the current working directory
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands

