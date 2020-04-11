## Prerequisites

Clone the repository:

```
$ git clone https://github.com/operator-framework/operator-sdk/
```

The docs are built with [Hugo]() which can be installed along with the
required extensions by following the [docsy install
guide](https://www.docsy.dev/docs/getting-started/#prerequisites-and-installation).

We use `git submodules` to install the docsy theme. From the
`operator-sdk` directory, update the submodules to install the theme.

```
$ git submodule update --init --recursive
```

## Build and Serve

You can build and serve your docs to localhost:1313. From the `website`
directory run:

```
hugo server
```

Any changes will be included in real time.
