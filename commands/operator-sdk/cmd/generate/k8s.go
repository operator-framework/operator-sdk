package generate

import (
	"fmt"

	cmdError "github.com/coreos/operator-sdk/commands/operator-sdk/error"
	"github.com/spf13/cobra"
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
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("k8s command doesn't accept any inputs."))
	}
}
