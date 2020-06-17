// Copyright 2020 The Operator-SDK Authors
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
package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

type createAPIPlugin struct {
	config *config.Config

	// For help text.
	commandName string

	// Helm APIFlags
	apiFlags APIFlags
}

var (
	_ plugin.CreateAPI = &createAPIPlugin{}
)

func (p *createAPIPlugin) UpdateContext(ctx *plugin.Context) {
	ctx.Description = `Create a Kubernetes API by creating a CR and CRD with the Helm Chart package directories.`
	p.commandName = ctx.CommandName
}

func (p *createAPIPlugin) BindFlags(fs *pflag.FlagSet) {
	p.apiFlags.AddTo(fs)
}

func (p *createAPIPlugin) InjectConfig(c *config.Config) {
	p.config = c
}

func (p *createAPIPlugin) Run() error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := p.Scaffold(); err != nil {
		return err
	}
	return nil
}

func (p *createAPIPlugin) Validate() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error to get the current path: %v", err)
	}
	projectName := filepath.Base(dir)

	// Check if the project name is a valid k8s object name.
	if err := validation.IsDNS1123Label(strings.ToLower(projectName)); err != nil {
		return fmt.Errorf("project name (%s) is invalid: %v", projectName, err)
	}

	if err := p.apiFlags.Validate(); err != nil {
		return err
	}
	return nil
}

func (p *createAPIPlugin) Scaffold() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error to get the current path: %v", err)
	}

	cfg := input.Config{
		AbsProjectPath: filepath.Join(projutil.MustGetwd()),
		ProjectName:    filepath.Base(dir),
	}

	createOpts := helm.CreateChartOptions{
		ResourceAPIVersion: p.apiFlags.APIVersion,
		ResourceKind:       p.apiFlags.Kind,
		Chart:              p.apiFlags.HelmChartRef,
		Version:            p.apiFlags.HelmChartVersion,
		Repo:               p.apiFlags.HelmChartRepo,
		CRDVersion:         p.apiFlags.CRDVersion,
	}

	if err := helm.API(cfg, createOpts); err != nil {
		return err
	}
	return nil
}
