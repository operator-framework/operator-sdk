---
title: "operator-sdk alpha config-gen install-as-plugin"
---
## operator-sdk alpha config-gen install-as-plugin

Install config-gen as a kustomize plugin

### Synopsis

Write a script to /home/runner/.config/kustomize/plugin/kubebuilder.sigs.k8s.io/v1alpha1/kubebuilderconfiggen/KubebuilderConfigGen for kustomize to locate as a plugin.

```
operator-sdk alpha config-gen install-as-plugin [flags]
```

### Examples

```
kubebuilder alpha config-gen install-as-plugin
```

### Options

```
  -h, --help   help for install-as-plugin
```

### Options inherited from parent commands

```
      --plugins strings          plugin keys of the plugin to initialize the project with
      --project-version string   project version
      --verbose                  Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha config-gen](../operator-sdk_alpha_config-gen)	 - Generate configuration for controller-runtime based projects

