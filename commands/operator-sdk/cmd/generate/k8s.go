package generate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	cmdError "github.com/coreos/operator-sdk/commands/operator-sdk/error"

	"github.com/spf13/cobra"
)

const (
	k8sGenerated = "./tmp/codegen/update-generated.sh"
)

func NewGenerateK8SCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "k8s",
		Short: "Generates Kubernetes code for custom resource",
		Long: `k8s generator generates code for custom resource given the API spec
to comply with kube-API requirements.
`,
		Run: k8sFunc,
	}
}

func k8sFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("k8s command doesn't accept any arguments."))
	}

	kcmd := exec.Command(k8sGenerated)
	o, err := kcmd.CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to perform code-generation for CustomResources: (%v)", string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))
}
