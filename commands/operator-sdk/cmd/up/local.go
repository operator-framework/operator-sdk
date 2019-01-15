// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package up

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewLocalCmd - up local command to run an operator loccally
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
	upLocalCmd.Flags().StringVar(&operatorFlags, "operator-flags", "", "The flags that the operator needs. Example: \"--flag1 value1 --flag2=value2\"")
	upLocalCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the operator watches for changes.")
	upLocalCmd.Flags().StringVar(&ldFlags, "go-ldflags", "", "Set Go linker options")
	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeAnsible:
		ansibleOperatorFlags = aoflags.AddTo(upLocalCmd.Flags(), "(ansible operator)")
	case projutil.OperatorTypeHelm:
		helmOperatorFlags = hoflags.AddTo(upLocalCmd.Flags(), "(helm operator)")
	}
	return upLocalCmd
}

var (
	kubeConfig           string
	operatorFlags        string
	namespace            string
	ldFlags              string
	ansibleOperatorFlags *aoflags.AnsibleOperatorFlags
	helmOperatorFlags    *hoflags.HelmOperatorFlags
)

const (
	defaultConfigPath = ".kube/config"
)

func upLocalFunc(cmd *cobra.Command, args []string) {
	mustKubeConfig()

	log.Info("Running the operator locally.")

	switch projutil.GetOperatorType() {
	case projutil.OperatorTypeGo:
		projutil.MustInProjectRoot()
		upLocal()
	case projutil.OperatorTypeAnsible:
		upLocalAnsible()
	case projutil.OperatorTypeHelm:
		upLocalHelm()
	default:
		log.Fatal("Failed to determine operator type")
	}
}

// mustKubeConfig checks if the kubeconfig file exists.
func mustKubeConfig() {
	// if kubeConfig is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(kubeConfig) == 0 {
		usr, err := user.Current()
		if err != nil {
			log.Fatalf("Failed to determine user's home dir: (%v)", err)
		}
		kubeConfig = filepath.Join(usr.HomeDir, defaultConfigPath)
	}

	_, err := os.Stat(kubeConfig)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("Failed to find the kubeconfig file (%v): (%v)", kubeConfig, err)
	}
}

func upLocal() {
	args := []string{"run"}
	if ldFlags != "" {
		args = append(args, []string{"-ldflags", ldFlags}...)
	}
	args = append(args, filepath.Join(scaffold.ManagerDir, scaffold.CmdFile))
	if operatorFlags != "" {
		extraArgs := strings.Split(operatorFlags, " ")
		args = append(args, extraArgs...)
	}
	dc := exec.Command("go", args...)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		err := dc.Process.Kill()
		if err != nil {
			log.Fatalf("Failed to terminate the operator: (%v)", err)
		}
		os.Exit(0)
	}()
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, kubeConfig))
	dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, namespace))
	err := dc.Run()
	if err != nil {
		log.Fatalf("Failed to run operator locally: (%v)", err)
	}
}

func upLocalAnsible() {
	// Set the kubeconfig that the manager will be able to grab
	if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeConfig); err != nil {
		log.Fatalf("Failed to set %s environment variable: (%v)", k8sutil.KubeConfigEnvVar, err)
	}
	// Set the kubeconfig that the manager will be able to grab
	if namespace != "" {
		if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace); err != nil {
			log.Fatalf("Failed to set %s environment variable: (%v)", k8sutil.WatchNamespaceEnvVar, err)
		}
	}

	ansible.Run(ansibleOperatorFlags)
}

func upLocalHelm() {
	// Set the kubeconfig that the manager will be able to grab
	if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeConfig); err != nil {
		log.Fatalf("Failed to set %s environment variable: (%v)", k8sutil.KubeConfigEnvVar, err)
	}

	// Set the kubeconfig that the manager will be able to grab
	if namespace != "" {
		if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace); err != nil {
			log.Fatalf("Failed to set %s environment variable: (%v)", k8sutil.WatchNamespaceEnvVar, err)
		}
	}

	helm.Run(helmOperatorFlags)
}

func printVersion() {
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Infof("Version of operator-sdk: %v", sdkVersion.Version)
}
