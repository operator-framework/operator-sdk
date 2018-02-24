package generate

import "github.com/spf13/cobra"

func NewGenerateK8SCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "k8s",
		Short: "Generates Kubernetes code for custom resource",
		Long: `k8s generator generates code for custom resource given the API spec
to comply with kube-API requirements.
`,
		Run: func(cmd *cobra.Command, args []string) {
			panic("UNIMPLEMENTED")
		},
	}
}
