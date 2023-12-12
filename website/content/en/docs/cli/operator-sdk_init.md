---
title: "operator-sdk init"
---
## operator-sdk init

Initialize a new project

### Synopsis

Initialize a new project including the following files:
  - a "go.mod" with project dependencies
  - a "PROJECT" file that stores project configuration
  - a "Makefile" with several useful make targets for the project
  - several YAML files for project deployment under the "config" directory
  - a "cmd/main.go" file that creates the manager that will run the project controllers


```
operator-sdk init [flags]
```

### Examples

```
  # Initialize a new project with your domain and name in copyright
  operator-sdk init --plugins go/v4 --domain example.org --owner "Your name"

  # Initialize a new project defining a specific project version
  operator-sdk init --plugins go/v4 --project-version 3

```

### Options

```
      --domain string            domain for groups (default "my.domain")
      --fetch-deps               ensure dependencies are downloaded (default true)
  -h, --help                     help for init
      --license string           license to use to boilerplate, may be one of 'apache2', 'none' (default "apache2")
      --owner string             owner to add to the copyright
      --project-name string      name of this project
      --project-version string   project version (default "3")
      --repo string              name to use for go module (e.g., github.com/user/repo), defaults to the go package of the current working directory.
      --skip-go-version-check    if specified, skip checking the Go version
```

### Options inherited from parent commands

```
      --plugins strings   plugin keys to be used for this subcommand execution
      --verbose           Enable verbose logging
```

### SEE ALSO

* [operator-sdk](../operator-sdk)	 - 

