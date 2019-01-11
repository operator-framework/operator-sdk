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

	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm/client"
	"github.com/operator-framework/operator-sdk/pkg/helm/controller"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
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
		RunE: upLocalFunc,
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

func upLocalFunc(cmd *cobra.Command, args []string) error {
	mustKubeConfig()

	log.Info("Running the operator locally.")

	t := projutil.GetOperatorType()
	switch t {
	case projutil.OperatorTypeGo:
		projutil.MustInProjectRoot()
		return upLocal()
	case projutil.OperatorTypeAnsible:
		return upLocalAnsible()
	case projutil.OperatorTypeHelm:
		return upLocalHelm()
	}
	return fmt.Errorf("unknown operator type '%v'", t)
}

// mustKubeConfig exits if the kubeconfig file does not exist.
func mustKubeConfig() {
	// If kubeConfig is not specified, search for the default kubeconfig file
	// under the $HOME/.kube/config.
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

func upLocal() error {
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
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, kubeConfig))
	dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, namespace))
	if err := projutil.ExecCmd(dc); err != nil {
		return fmt.Errorf("failed to run operator locally: (%v)", err)
	}
	return nil
}

func upLocalAnsible() error {
	// Set the kubeconfig that the manager will be able to grab
	if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeConfig); err != nil {
		return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.KubeConfigEnvVar, err)
	}
	// Set the kubeconfig that the manager will be able to grab
	if namespace != "" {
		if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace); err != nil {
			return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.WatchNamespaceEnvVar, err)
		}
	}

	return ansible.Run(ansibleOperatorFlags)
}

func upLocalHelm() error {
	// Set the kubeconfig that the manager will be able to grab
	if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeConfig); err != nil {
		return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.KubeConfigEnvVar, err)
	}

	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		return err
	}

	// Create Tiller's storage backend and kubernetes client
	storageBackend := storage.Init(driver.NewMemory())
	tillerKubeClient, err := client.NewFromManager(mgr)
	if err != nil {
		return err
	}

	factories, err := release.NewManagerFactoriesFromFile(storageBackend, tillerKubeClient, helmOperatorFlags.WatchesFile)
	if err != nil {
		return err
	}

	for gvk, factory := range factories {
		// Register the controller with the factory.
		err := controller.Add(mgr, controller.WatchOptions{
			Namespace:       namespace,
			GVK:             gvk,
			ManagerFactory:  factory,
			ReconcilePeriod: helmOperatorFlags.ReconcilePeriod,
		})
		if err != nil {
			return err
		}
	}

	// Start the Cmd
	return mgr.Start(signals.SetupSignalHandler())
}

func printVersion() {
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Infof("Version of operator-sdk: %v", sdkVersion.Version)
}
