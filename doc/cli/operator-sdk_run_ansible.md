## operator-sdk run ansible

Runs as an ansible operator

### Synopsis

Runs as an ansible operator. This is intended to be used when running
in a Pod inside a cluster. Developers wanting to run their operator locally
should use "up local" instead.

```
operator-sdk run ansible [flags]
```

### Options

```
      --ansible-verbosity int            Ansible verbosity. Overridden by environment variable. (default 2)
  -h, --help                             help for ansible
      --inject-owner-ref                 The ansible operator will inject owner references unless this flag is false (default true)
      --max-workers int                  Maximum number of workers to use. Overridden by environment variable. (default 1)
      --reconcile-period duration        Default reconcile period for controllers (default 1m0s)
      --watches-file string              Path to the watches file to use (default "./watches.yaml")
      --zap-devel                        Enable zap development mode (changes defaults to console encoder, debug log level, and disables sampling)
      --zap-encoder encoder              Zap log encoding ('json' or 'console')
      --zap-level level                  Zap log level (one of 'debug', 'info', 'error' or any integer value > 0) (default info)
      --zap-sample sample                Enable zap log sampling. Sampling will be disabled for integer log levels > 1
      --zap-time-encoding timeEncoding   Sets the zap time format ('epoch', 'millis', 'nano', or 'iso8601') (default )
```

### SEE ALSO

* [operator-sdk run](operator-sdk_run.md)	 - Runs a generic operator

