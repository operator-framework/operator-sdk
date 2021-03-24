---
title: "operator-sdk completion zsh"
---
## operator-sdk completion zsh

Load zsh completions

```
operator-sdk completion zsh [flags]
```

### Examples

```
# If shell completion is not already enabled in your environment you will need
# to enable it. You can execute the following once:
$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ operator-sdk completion zsh > "${fpath[1]}/_operator-sdk"

# You will need to start a new shell for this setup to take effect.

```

### Options

```
  -h, --help   help for zsh
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk completion](../operator-sdk_completion)	 - Load completions for the specified shell

