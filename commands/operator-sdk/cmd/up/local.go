package up

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	"github.com/spf13/cobra"
)

func NewLocalCmd() *cobra.Command {
	upLocalCmd := &cobra.Command{
		Use:   "local",
		Short: "Launches the operator locally",
		Long: `The operator-sdk up local command launches the operator on the local machine 
by building the operator binary with the ability to access a 
kubernetes cluster using a kubeconfig file.
`,
		Run: upLocalFunc,
	}

	upLocalCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")

	return upLocalCmd
}

var (
	kubeConfig string
)

const (
	gocmd             = "go"
	run               = "run"
	cmd               = "cmd"
	main              = "main.go"
	defaultConfigPath = ".kube/config"
)

func upLocalFunc(cmd *cobra.Command, args []string) {
	mustKubeConfig()
	cmdutil.MustInProjectRoot()
	c := cmdutil.GetConfig()
	upLocal(c.ProjectName)
}

// mustKubeConfig checks if the kubeconfig file exists.
func mustKubeConfig() {
	// if kubeConfig is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(kubeConfig) == 0 {
		usr, err := user.Current()
		if err != nil {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to determine user's home dir: %v", err))
		}
		kubeConfig = filepath.Join(usr.HomeDir, defaultConfigPath)
	}

	_, err := os.Stat(kubeConfig)
	if err != nil && os.IsNotExist(err) {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to find the kubeconfig file (%v): %v", kubeConfig, err))
	}
}

func upLocal(projectName string) {
	dc := exec.Command(gocmd, run, filepath.Join(cmd, projectName, main))
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, kubeConfig))
	err := dc.Run()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to run operator locally: %v", err))
	}
}
