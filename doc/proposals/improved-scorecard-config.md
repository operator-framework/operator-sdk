# Improved Scorecard Config

Implementation Owner: AlexNPavel
Status: Draft

- [Background](#Background)
- [Goals](#Goals)
- [Design overview](#Design-overview)
- [User facing usage](#User-facing-usage)

## Background

The `scorecard` subcommand in the `operator-sdk` currently has ~15 different command line flags. This creates
unnecessary complexity for users. Also, many of the config options only apply to the internal plugins, and
this may become confusing to users who use external plugins, as they may think that these config options
affect their external plugins. Using a config file that configures plugins on an individual
basis can help clarify these differences as well as make configuration a lot cleaner

## Goals

Drastically reduce the amount of command line flags in the `scorecard` subcommand and define a configuration file
format that can allow for the simple configuration of both internal and external plugins.

## Design overview

The configuration for the scorecard will be in a `scorecard` subsection of the config file. This will allow
the config file to continue working properly when the SDK adds global config file support for all subcommands.
Under the scorecard subsection there are configuration options that apply to the entire scorecard as well as
a section that allows for the configuration of both internal and external plugins. Here are the config options
and what they do:

- `kubeconfig` string - path to kubeconfig. This option sets the kubeconfig for internal plugins and sets the `KUBECONFIG` env var for external plugins
- `output` string - sets output format. Valid values: `text` and `json`
- `plugin-dir` string - path to scorecard plugin directory. This is the directory where the plugins are run from, and all executable files in a `bin` subdirectory of the `plugin-dir` are automtically run by default.
- `plugins` - an array of objects that configure both internal and external scorecard plugins

The objects in the `plugins` configuration have 3 elements: `name`, `disable`, and either `basic`, `olm`, or `external`. The `name` is
the name of the plugin and `basic`, `olm`, and `external` are configuration blocks used if the plugin is the `basic`, `olm`, or an `external`
plugin respectively. If any of `basic`, `olm`, or `external` are specified for the same plugin, the plugin is automatically marked as failed. The
`disable` field is `false` by default, but can be set to `true` to disable tests that would be automatically run, such as the
`basic` and `olm` tests and external plugins in `{plugin-dir}/bin`. For internal plugins, the `basic` or `olm` struct must be set in
to identify the plugin and for external plugins the `command` field must be set in the `external` config block for the `disable`
config to work properly.

Configuration for both `basic` and `olm` contains all of the original configuration options for the scorecard that pertained to internal plugins. They are:

- `namespace`
- `init-timeout`
- `olm-deployed`
- `csv-path`
- `namespaced-manifest`
- `global-manifest`
- `cr-manifest`
- `proxy-image`
- `proxy-pull-policy`
- `crds-dir`

Configuration for `external` contains 3 fields:

- `command` string - path to the command being run. Can be relative or absolute. If an executable from `{plugin-dir}/bin` is specified in the `command` field, it will not be automatically run without configuration as it would be otherwise. The same command can be specified in multiple plugins if a user wishes to run the same plugin multiple times with different configurations.
- `args` \[\]string - a string array consisting of args passed to the command
- `env` - array of elements that contain 2 fields, `name` and `value`, that configure the environment variables that the command is run with. If a user specified a kubeconfig in the main `scorecard` config and also sets `KUBECONFIG` in this section, the `KUBECONFIG` environment variable in this section has priority. This allows a user to run certain plugins under a different Kubernetes environment if necessary

Here is an example config:

```yaml
scorecard:
  output: json
  plugins:
    - name: Basic Tests
      basic:
        cr-manifest:
          - "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
          - "deploy/crds/cache_v1alpha1_memcachedrs_cr.yaml"
        init-timeout: 60
        csv-path: "deploy/olm-catalog/memcached-operator/0.0.3/memcached-operator.v0.0.3.clusterserviceversion.yaml"
        proxy-image: "scorecard-proxy"
        proxy-pull-policy: "Never"
    - name: OLM Tests
      olm:
        cr-manifest:
          - "deploy/crds/cache_v1alpha1_memcached_cr.yaml"
          - "deploy/crds/cache_v1alpha1_memcachedrs_cr.yaml"
        init-timeout: 60
        csv-path: "deploy/olm-catalog/memcached-operator/0.0.3/memcached-operator.v0.0.3.clusterserviceversion.yaml"
        proxy-image: "scorecard-proxy"
        proxy-pull-policy: "Never"
    - name: Custom Test
      external:
        command: bin/my-test.sh
    - name: Custom Test v2
      external:
        command: bin/my-test.sh
        args: ["--version=2"]
    - name: Custom Test Cluster 2
      external:
        command: bin/my-test.sh
        env:
          - name: KUBECONFIG
            value: "~/.kube/config2`
```

## User facing usage

This change would be a pretty large breaking change for users. The only flag that would remain in the scorecard would be the
`config` flag to allow users to specify where their config file is. We would need to make sure we have updated documentation
for scorecard users ready at the same time that we merge these changes to reduce confusion and help users migrate to the
new configuration format.
