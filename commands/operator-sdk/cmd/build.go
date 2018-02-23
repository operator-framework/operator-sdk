package cmd

import "github.com/spf13/cobra"

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build <image>",
	Short: "Compiles code and builds artifacts",
	Long: `The operator-sdk build command compiles the code, builds the executables,
	and generates Kubernetes manifests.

	<image> is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".
	This image will be automatically set in the deployment manifests.

	After build completes, the image would be built locally in docker. Then it needs to
	be pushed to remote registry.
	For example:
	$ operator-sdk build quay.io/example/operator:v0.0.1
	$ docker push quay.io/example/operator:v0.0.1
`,
	Run: newFunc,
}
