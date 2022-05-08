---
title: "operator-sdk alpha config-gen"
---
## operator-sdk alpha config-gen

Generate configuration for controller-runtime based projects

### Synopsis

config-gen programatically generates configuration for a controller-runtime based
project using the project source code (golang) and a KubebuilderConfigGen resource file.

This is an alternative to expressing configuration as a static set of kustomize patches
in the "config" directory.

config-gen may be used as a standalone command run against a file, as a kustomize
transformer plugin, or as a configuration function (e.g. kpt).

config-gen uses the controller-tools generators to generate CRDs from the go source
and then generates additional resources such as the namespace, controller-manager,
webhooks, etc.

Following is an example KubebuilderConfigGen resource used by config-gen:

  # kubebuilderconfiggen.yaml
  # this resource describes how to generate configuration for a controller-runtime
  # based project
  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: my-project-name
  spec:
    controllerManager:
      image: my-org-name/my-project-name:v0.1.0

If this file was at the project source root, config-gen could be used to emit
configuration using:

  kubebuilder alpha config-gen ./kubebuilderconfiggen.yaml

The KubebuilderConfigGen resource has the following fields:

  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen

  metadata:
    # name of the project.  used in various resource names.
    # required
    name: project-name

    # namespace for the project
    # optional -- defaults to "${metadata.name}-system"
    namespace: project-namespace

  spec:
    # configure how CRDs are generated
    crds:
      # path to go module source directory provided to controller-gen libraries
      # optional -- defaults to '.'
      sourceDirectory: ./relative/path

    # configure how the controller-manager is generated
    controllerManager:
      # image to run
      image: my-org/my-project:v0.1.0

      # if set, use component config for the controller-manager
      # optional
      componentConfig:
        # use component config
        enable: true

        # path to component config to put into a ConfigMap
        configFilepath: ./path/to/componentconfig.yaml

      # configure how metrics are exposed
      metrics:
        # disable the auth proxy required for scraping metrics
        # disable: false

        # generate prometheus ServiceMonitor resource
        enableServiceMonitor: true

    # configure how webhooks are generated
    # optional -- defaults to not generating webhook configuration
    webhooks:
      # enable will cause webhook config to be generated
      enable: true

      # configures crds which use conversion webhooks
      enableConversion:
        # key is the name of the CRD
        "bars.example.my.domain": true

      # configures where to get the certificate used for webhooks
      # discriminated union
      certificateSource:
        # type of certificate source
        # one of ["certManager", "dev", "manual"] -- defaults to "manual"
        # certManager: certmanager is used to manage certificates -- requires CertManager to be installed
        # dev: certificate is generated and wired into resources
        # manual: no certificate is generated or wired into resources
        type: "dev"

        # options for a dev certificate -- requires "dev" as the type
        devCertificate:
          duration: 1h

```
operator-sdk alpha config-gen PROJECT_FILE [RESOURCE_PATCHES...] [flags]
```

### Examples

```
#
# As command
#
# create the kubebuilderconfiggen.yaml at project root
cat > kubebuilderconfiggen.yaml <<EOF
apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: project
  spec:
    controllerManager
      image: org/project:v0.1.0
EOF

# run the config generator
kubebuilder alpha config-gen kubebuilderconfiggen.yaml

# run the config generator and apply
kubebuilder alpha config-gen kubebuilderconfiggen.yaml | kubectl apply -f -

# generate configuration from a file with patches
kubebuilder alpha config-gen kubebuilderconfiggen.yaml patch1.yaml patch2.yaml

#
# As Kustomize plugin
# this allows using config-gen with kustomize features such as patches, commonLabels,
# commonAnnotations, resources, configMapGenerator and other transformer plugins.
#

# install the kustomize version used in the v3 plugin
# set VERSION to install a different version
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/kustomize/v${VERSION:-3.8.9}/hack/install_kustomize.sh" | bash -s -- "${VERSION:-3.8.9}"

# install the command as a kustomize plugin
kubebuilder alpha config-gen install-as-plugin

# create the kustomization.yaml containing the KubebuilderConfigGen resource
cat > kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
transformers:
- |-
  apiVersion: kubebuilder.sigs.k8s.io/v1alpha1
  kind: KubebuilderConfigGen
  metadata:
    name: my-project
  spec:
    controllerManager:
      image: my-org/my-project:v0.1.0
EOF

# generate configuration from kustomize > v4.0.0
kustomize build --enable-alpha-plugins .

# generate configuration from kustomize <= v4.0.0
kustomize build --enable_alpha_plugins .
```

### Options

```
  -h, --help    help for config-gen
      --stack   print the stack trace on failure
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk alpha](../operator-sdk_alpha)	 - Alpha-stage subcommands
* [operator-sdk alpha config-gen install-as-plugin](../operator-sdk_alpha_config-gen_install-as-plugin)	 - Install config-gen as a kustomize plugin

