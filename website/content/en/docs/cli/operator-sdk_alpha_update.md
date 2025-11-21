---
title: "operator-sdk alpha update"
---
## operator-sdk alpha update

Update your project to a newer version (3-way merge; squash by default)

### Synopsis

Upgrade your project scaffold using a 3-way merge while preserving your code.

The updater uses four temporary branches during the run:
  • ancestor : clean scaffold from the starting version (--from-version)
  • original : snapshot of your current project (--from-branch)
  • upgrade  : scaffold generated with the target version (--to-version)
  • merge    : result of merging original into upgrade (conflicts possible)

Output branch & history:
  • Default: SQUASH the merge result into ONE commit on:
        kubebuilder-update-from-&lt;from-version&gt;-to-&lt;to-version&gt;
  • --show-commits: keep full history (not compatible with --restore-path).

Conflicts:
  • Default: stop on conflicts and leave the merge branch for manual resolution.
  • --force: commit with conflict markers so automation can proceed.

Other options:
  • --restore-path: restore paths from base when squashing (e.g., CI configs).
  • --output-branch: override the output branch name.
  • --push: push the output branch to 'origin' after the update.
  • --git-config: pass per-invocation Git config as -c key=value (repeatable). When not set,
      defaults are set to improve detection during merges.

Defaults:
  • --from-version / --to-version: resolved from PROJECT and the latest release if unset.
  • --from-branch: defaults to 'main' if not specified.

```
operator-sdk alpha update [flags]
```

### Examples

```

  # Update from the version in PROJECT to the latest, stop on conflicts
  kubebuilder alpha update

  # Update from a specific version to latest
  kubebuilder alpha update --from-version v4.6.0

  # Update from v4.5.0 to v4.7.0 and keep conflict markers (automation-friendly)
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force

  # Keep full commit history instead of squashing
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force --show-commits

  # Squash while preserving CI workflows from base (e.g., main)
  kubebuilder alpha update --force --restore-path .github/workflows

  # Show commits into a custom output branch name
  kubebuilder alpha update --force --show-commits --output-branch my-update-branch

  # Run update and push the output branch to origin (works with or without --show-commits)
  kubebuilder alpha update --from-version v4.6.0 --to-version v4.7.0 --force --push

  # Create an issue and add an AI overview comment
  kubebuilder alpha update --open-gh-issue --use-gh-models

  # Add extra Git configs (no need to re-specify defaults)
  kubebuilder alpha update --git-config merge.conflictStyle=diff3 --git-config rerere.enabled=true
                                          
  # Disable Git config defaults completely, use only custom configs
  kubebuilder alpha update --git-config disable --git-config rerere.enabled=true
```

### Options

```
      --force                             Force the update even if conflicts occur. Conflicted files will include conflict markers, and a commit will be created automatically. Ideal for automation (e.g., cronjobs, CI).
      --from-branch string                Git branch to use as current state of the project for the update.
      --from-version string               binary release version to upgrade from. Should match the version used to init the project and be a valid release version, e.g., v4.6.0. If not set, it defaults to the version specified in the PROJECT file.
      --git-config --git-config disable   Per-invocation Git config (repeatable). Defaults: -c merge.renameLimit=999999 -c diff.renameLimit=999999 -c merge.conflictStyle=merge. Your configs are applied on top. To disable defaults, include --git-config disable
  -h, --help                              help for update
      --open-gh-issue gh                  Create a GitHub issue with a pre-filled checklist and compare link after the update completes (requires gh).
      --output-branch string              Override the default output branch name (default: kubebuilder-update-from-<from-version>-to-<to-version>).
      --push                              Push the output branch to the remote repository after the update.
      --restore-path stringArray          Paths to preserve from the base branch (repeatable). Not supported with --show-commits.
      --show-commits                      If set, the update will keep the full history instead of squashing into a single commit.
      --to-version string                 binary release version to upgrade to. Should be a valid release version, e.g., v4.7.0. If not set, it defaults to the latest release version available in the project repository.
      --use-gh-models gh models run       Generate and post an AI summary comment to the GitHub Issue using gh models run. Requires --open-gh-issue and GitHub CLI (`gh`) with the `gh-models` extension.
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands

