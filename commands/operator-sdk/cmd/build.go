package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewBuildCmd() *cobra.Command {
	return &cobra.Command{
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
		Run: buildFunc,
	}
}

const (
	build       = "./tmp/build/build.sh"
	dockerBuild = "./tmp/build/docker_build.sh"
)

func buildFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("new command needs 1 argument."))
	}

	bcmd := exec.Command(build)
	o, err := bcmd.CombinedOutput()
	if err != nil {
		ExitWithError(ExitError, fmt.Errorf("failed to build: (%v)", string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))

	image := args[0]
	dbcmd := exec.Command(dockerBuild)
	dbcmd.Env = append(os.Environ(), fmt.Sprintf("IMAGE=%v", image))
	o, err = dbcmd.CombinedOutput()
	if err != nil {
		ExitWithError(ExitError, fmt.Errorf("failed to output build image %v: (%v)", image, string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))
	// TODO: generates Kubernetes manifests
}
