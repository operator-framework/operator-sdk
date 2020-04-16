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

package golang

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

type initPlugin struct {
	plugin.Init

	config *config.Config
}

var _ plugin.Init = &initPlugin{}

func (p *initPlugin) UpdateContext(ctx *plugin.Context) { p.Init.UpdateContext(ctx) }
func (p *initPlugin) BindFlags(fs *pflag.FlagSet)       { p.Init.BindFlags(fs) }

func (p *initPlugin) InjectConfig(c *config.Config) {
	p.Init.InjectConfig(c)
	p.config = c
}

func (p *initPlugin) Run() error {
	if err := p.Init.Run(); err != nil {
		return err
	}

	if err := initUpdateMakefile(); err != nil {
		return fmt.Errorf("error updating Makefile: %v", err)
	}

	// Update plugin config section with this plugin's configuration.
	cfg := Config{}
	if err := p.config.EncodePluginConfig(pluginConfigKey, cfg); err != nil {
		return fmt.Errorf("error writing plugin config for %s: %v", pluginConfigKey, err)
	}

	return nil
}

const (
	makefileBundleImgVarFragment = `# Current operator version
VERSION ?= 0.0.1
# Bundle image URL
BUNDLE_IMG ?= controller-bundle:$(VERSION)`

	makefileManifestsName = "manifests"
	//nolint:lll
	makefileManifestsFragment = `manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	operator-sdk generate bundle -q --kustomize
`

	makefileBundleName     = "bundle"
	makefileBundleFragment = `# Generate a bundle directory
bundle: manifests
	kustomize build config/bundle | operator-sdk generate bundle -q --manifests --version $(VERSION)
`

	makefileBuildBundleName = "bundle-build"
	//nolint:lll
	makefileBuildBundleFragment = `# Build the bundle OCI image
bundle-build: manifests
	kustomize build config/bundle | operator-sdk generate bundle -q --manifests --metadata --overwrite --version $(VERSION)
	operator-sdk bundle validate config/bundle
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
`
)

func initUpdateMakefile() error {
	makefileBytes, err := ioutil.ReadFile("Makefile")
	if err != nil {
		return err
	}

	makefileBytes = append([]byte(makefileBundleImgVarFragment), makefileBytes...)

	// Modify Makefile with OLM recipes.
	namedFragments := map[string]string{
		makefileManifestsName:   makefileManifestsFragment,
		makefileBundleName:      makefileBundleFragment,
		makefileBuildBundleName: makefileBuildBundleFragment,
	}
	for name, fragment := range namedFragments {
		makefileBytes = replaceOrAppendMakefileRecipe(makefileBytes, name, fragment)
	}

	return ioutil.WriteFile("Makefile", makefileBytes, 0644)
}

func replaceOrAppendMakefileRecipe(oldMakefile []byte, recipeName, fragment string) (newMakefile []byte) {
	// Clean up fragment.
	fragment = strings.TrimSpace(fragment) + "\n"

	// TODO: handle comments above recipes.
	var foundRecipeStart, foundRecipeEnd bool
	scanner := bufio.NewScanner(bytes.NewBuffer(oldMakefile))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, recipeName+":"):
			foundRecipeStart = true
			newMakefile = append(newMakefile, []byte(fragment+"\n")...)
		case isRecipeLine(line) && foundRecipeStart:
			foundRecipeEnd = true
			fallthrough
		case !foundRecipeStart || foundRecipeEnd:
			newMakefile = append(newMakefile, []byte(line+"\n")...)
		}
	}

	if !foundRecipeStart {
		newMakefile = append(newMakefile, "\n"+fragment...)
	}

	return newMakefile
}

func isRecipeLine(line string) bool {
	recipeNameEnd := strings.Index(line, ":")
	return recipeNameEnd != -1 && !strings.Contains(line[:recipeNameEnd], " ")
}
