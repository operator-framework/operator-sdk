---
title: "operator-sdk completion bash"
---
## operator-sdk completion bash

Load bash completions

```
operator-sdk completion bash [flags]
```

### Examples

```
# To load completion for this session, execute:
$ source <(operator-sdk completion bash)

# To load completions for each session, execute once:
Linux:
  $ operator-sdk completion bash > /etc/bash_completion.d/operator-sdk
MacOS:
  $ operator-sdk completion bash > /usr/local/etc/bash_completion.d/operator-sdk

```

### Options

```
  -h, --help   help for bash
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk completion](../operator-sdk_completion)	 - Load completions for the specified shell

