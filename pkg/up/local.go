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
	"strings"
	"syscall"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type UpLocalCmd struct {
	KubeconfigPath string
	OperatorFlags  string
	Namespace      string
	LDFlags        string
}

func (c *UpLocalCmd) Run() error {
	projutil.MustInProjectRoot()

	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)
	outputBinName := filepath.Join(scaffold.BuildBinDir, projectName+"-local")
	if err := buildLocal(viper.GetString(config.RepoOpt), outputBinName, c.LDFlags); err != nil {
		return fmt.Errorf("failed to build operator to run locally: (%v)", err)
	}

	args := []string{}
	if c.OperatorFlags != "" {
		extraArgs := strings.Split(c.OperatorFlags, " ")
		args = append(args, extraArgs...)
	}
	dc := exec.Command(outputBinName, args...)
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		err := dc.Process.Kill()
		if err != nil {
			log.Fatalf("Failed to terminate the operator: (%v)", err)
		}
		os.Exit(0)
	}()
	dc.Env = os.Environ()
	// only set env var if user explicitly specified a kubeconfig path
	if c.KubeconfigPath != "" {
		dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.KubeConfigEnvVar, c.KubeconfigPath))
	}
	dc.Env = append(dc.Env, fmt.Sprintf("%v=%v", k8sutil.WatchNamespaceEnvVar, c.Namespace))
	if err := projutil.ExecCmd(dc); err != nil {
		return fmt.Errorf("failed to run operator locally: (%v)", err)
	}
	return nil
}

func SetupOperatorEnv(kubeconfigPath, namespace string) error {
	// Set the kubeconfig that the manager will be able to grab
	// only set env var if user explicitly specified a kubeconfig path
	if kubeconfigPath != "" {
		if err := os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfigPath); err != nil {
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

func buildLocal(repo, outputBinName, ldFlags string) error {
	var args []string
	if ldFlags != "" {
		args = []string{"-ldflags", ldFlags}
	}
	opts := projutil.GoCmdOptions{
		BinName:     outputBinName,
		PackagePath: filepath.Join(repo, scaffold.ManagerDir),
		Args:        args,
		GoMod:       projutil.IsDepManagerGoMod(),
	}
	return projutil.GoBuild(opts)
}
