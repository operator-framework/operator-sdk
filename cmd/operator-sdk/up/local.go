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
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible"
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/helm"
	hoflags "github.com/operator-framework/operator-sdk/pkg/helm/flags"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// newLocalCmd - up local command to run an operator loccally
func newLocalCmd() *cobra.Command {
	upLocalCmd := &cobra.Command{
		Use:   "local",
		Short: "Launches the operator locally",
		Long: `The operator-sdk up local command launches the operator on the local machine
by building the operator binary with the ability to access a
kubernetes cluster using a kubeconfig file.
`,
		RunE: upLocalFunc,
	}

	upLocalCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to location specified by $KUBECONFIG with a fallback to $HOME/.kube/config if not set")
	upLocalCmd.Flags().StringVar(&operatorFlags, "operator-flags", "", "The flags that the operator needs. Example: \"--flag1 value1 --flag2=value2\"")
	upLocalCmd.Flags().StringVar(&namespace, "namespace", "", "The namespace where the operator watches for changes.")
	upLocalCmd.Flags().StringVar(&ldFlags, "go-ldflags", "", "Set Go linker options")
	upLocalCmd.Flags().BoolVar(&disableAutoReload, "disable-auto-reload", false, "Disable auto reload of the operator for sources code changes.")
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
	disableAutoReload    bool
	ansibleOperatorFlags *aoflags.AnsibleOperatorFlags
	helmOperatorFlags    *hoflags.HelmOperatorFlags
)

func upLocalFunc(cmd *cobra.Command, args []string) error {
	log.Info("Running the operator locally.")

	// get default namespace to watch if unset
	if !cmd.Flags().Changed("namespace") {
		_, defaultNamespace, err := k8sInternal.GetKubeconfigAndNamespace(kubeConfig)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig and default namespace: %v", err)
		}
		namespace = defaultNamespace
	}
	log.Infof("Using namespace %s.", namespace)

	switch t := projutil.GetOperatorType(); t {
	case projutil.OperatorTypeGo:
		return upLocal()
	case projutil.OperatorTypeAnsible:
		return upLocalAnsible()
	case projutil.OperatorTypeHelm:
		return upLocalHelm()
	}
	return projutil.ErrUnknownOperatorType{}
}

func upLocal() error {
	log.Debug("Running Go operator")
	projutil.MustInProjectRoot()
	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)
	outputBinName := filepath.Join(scaffold.BuildBinDir, projectName+"-local")
	if err := buildLocal(outputBinName); err != nil {
		return fmt.Errorf("failed to build operator to run locally: (%v)", err)
	}

	args := []string{}
	if operatorFlags != "" {
		extraArgs := strings.Split(operatorFlags, " ")
		args = append(args, extraArgs...)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to start the Filesystem watcher: (%v)", err)
	}
	defer watcher.Close()

	dc := exec.Command(outputBinName, args...)
	dc.Env = os.Environ()
	// only set env var if user explicitly specified a kubeconfig path
	if kubeConfig != "" {
		dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, kubeConfig))
	}
	dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, namespace))
	if err := projutil.StartCmd(dc); err != nil {
		log.Fatalf("Failed to run operator locally: (%v)", err)
	}

	closeChannel := make(chan os.Signal)
	signal.Notify(closeChannel, os.Interrupt, syscall.SIGTERM)

	if disableAutoReload {
		go func() {
			<-closeChannel
			log.Debug("Received interrupt signal")
			if err := dc.Process.Kill(); err != nil {
				log.Fatalf("Failed to terminate the operator: (%v)", err)
			}
			os.Exit(0)
		}()
		if err := dc.Wait(); err != nil {
			log.Fatalf("Failed to run and exit operator locally: (%v)", err)

		}
		return nil
	}

	done := make(chan bool)
	go func(dc *exec.Cmd) {
		for {
			select {
			case event, ok := <-watcher.Events:
				log.Debugf("Event triggered: (%v)", event)
				if !ok {
					log.Fatal("Filesystem watcher channel for events closed")
				}
				if !strings.HasSuffix(event.Name, ".go") {
					continue
				}
				if err := dc.Process.Kill(); err != nil {
					log.Fatalf("Failed to terminate the operator: (%v)", err)
				}
				dc = exec.Command(outputBinName, args...)
				dc.Env = os.Environ()
				// only set env var if user explicitly specified a kubeconfig path
				if kubeConfig != "" {
					dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, kubeConfig))
				}
				dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, namespace))

				if err := buildLocal(outputBinName); err != nil {
					log.Fatalf("Failed to build operator to run locally: (%v)", err)
				}
				if err := projutil.StartCmd(dc); err != nil {
					log.Fatalf("Failed to run operator locally: (%v)", err)
				}
			case err, ok := <-watcher.Errors:
				log.Debugf("Filesystem watcher error: (%v)", err)
				if !ok {
					log.Fatal("Filesystem watcher channel for events closed")
				}
			case <-closeChannel:
				log.Debug("Received interrupt signal")
				if err := dc.Process.Kill(); err != nil {
					log.Fatalf("Failed to terminate the operator: (%v)", err)
				}
				os.Exit(0)
			}
		}
	}(dc)

	err = addRecursiveWatch(watcher, absProjectPath)
	if err != nil {
		log.Fatalf("Failed to watch directory recursively: (%v)", err)
	}
	<-done

	return nil
}

// addRecursiveWatch handles adding watches recursively for the path provided
// and its subdirectories.  If a non-directory is specified, this call is a no-op.
// Taken from https://github.com/openshift/origin/blob/85eb37b34f0657631592356d020cef5a58470f8e/pkg/util/fsnotification/fsnotification.go#L16
func addRecursiveWatch(watcher *fsnotify.Watcher, path string) error {
	file, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error introspecting path %s: %v", path, err)
	}
	if !file.IsDir() {
		return nil
	}

	folders, err := getSubFolders(path)
	for _, v := range folders {
		if strings.HasPrefix(v, path+"/.git") {
			continue
		}
		if strings.HasPrefix(v, path+"/build") {
			continue
		}
		//log.Debugf("Adding watch on directory: %s", v)
		err = watcher.Add(v)
		if err != nil {
			// "no space left on device" issues are usually resolved via
			// $ sudo sysctl fs.inotify.max_user_watches=65536
			return fmt.Errorf("error adding watcher for path %s: %v", v, err)
		}
	}
	return nil
}

// getSubFolders recursively retrieves all subfolders of the specified path.
func getSubFolders(path string) (paths []string, err error) {
	err = filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			paths = append(paths, newPath)
		}
		return nil
	})
	return paths, err
}

func upLocalAnsible() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(); err != nil {
		return err
	}
	return ansible.Run(ansibleOperatorFlags)
}

func upLocalHelm() error {
	logf.SetLogger(zap.Logger())
	if err := setupOperatorEnv(); err != nil {
		return err
	}
	return helm.Run(helmOperatorFlags)
}

func setupOperatorEnv() error {
	// Set the kubeconfig that the manager will be able to grab
	// only set env var if user explicitly specified a kubeconfig path
	if kubeConfig != "" {
		if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeConfig); err != nil {
			return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.KubeConfigEnvVar, err)
		}
	}
	// Set the namespace that the manager will be able to grab
	if namespace != "" {
		if err := os.Setenv(k8sutil.WatchNamespaceEnvVar, namespace); err != nil {
			return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.WatchNamespaceEnvVar, err)
		}
	}
	// Set the operator name, if not already set
	projutil.MustInProjectRoot()
	if _, err := k8sutil.GetOperatorName(); err != nil {
		operatorName := filepath.Base(projutil.MustGetwd())
		if err := os.Setenv(k8sutil.OperatorNameEnvVar, operatorName); err != nil {
			return fmt.Errorf("failed to set %s environment variable: (%v)", k8sutil.OperatorNameEnvVar, err)
		}
	}
	return nil
}

func buildLocal(outputBinName string) error {
	var args []string
	if ldFlags != "" {
		args = []string{"-ldflags", ldFlags}
	}
	opts := projutil.GoCmdOptions{
		BinName:     outputBinName,
		PackagePath: filepath.Join(projutil.CheckAndGetProjectGoPkg(), scaffold.ManagerDir),
		Args:        args,
		GoMod:       projutil.IsDepManagerGoMod(),
	}
	return projutil.GoBuild(opts)
}

func printVersion() {
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Infof("Version of operator-sdk: %v", sdkVersion.Version)
}
