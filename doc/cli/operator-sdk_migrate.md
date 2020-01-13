## operator-sdk migrate

Adds source code to an operator

### Synopsis

operator-sdk migrate adds a main.go source file and any associated source files for an operator that is not of the "go" type.

```
operator-sdk migrate [flags]
```

### Options

```
      --header-file string   Path to file containing headers for generated Go files. Copied to hack/boilerplate.go.txt
  -h, --help                 help for migrate
      --repo string          Project repository path. Used as the project's Go import path. This must be set if outside of $GOPATH/src (e.g. github.com/example-inc/my-operator)
```

### SEE ALSO

* [operator-sdk](operator-sdk.md)	 - An SDK for building operators with ease

