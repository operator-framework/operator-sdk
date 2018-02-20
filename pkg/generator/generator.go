package generator

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultFileMode = 0750
	cmdDir          = "cmd"
	deployDir       = "deploy"
	configDir       = "config"
	tmpDir          = "tmp"
	buildDir        = tmpDir + "/build"
	codegenDir      = tmpDir + "/codegen"
	pkgDir          = "pkg"
	apisDir         = pkgDir + "/apis"
	stubDir         = pkgDir + "/stub"
)

type Generator struct {
	apiGroup string
	kind     string
	// projectName is name of the new operator application
	// and is also the name of the base directory.
	projectName string
}

// NewGenerator creates a new scaffold Generator.
func NewGenerator(apiGroup, kind, projectName string) *Generator {
	return &Generator{apiGroup: apiGroup, kind: kind, projectName: projectName}
}

// Render generates the default project structure.
func (g *Generator) Render() error {
	if err := g.renderCmd(); err != nil {
		return err
	}
	if err := g.renderConfig(); err != nil {
		return err
	}
	if err := g.renderDeploy(); err != nil {
		return err
	}
	if err := g.renderPkg(); err != nil {
		return err
	}
	return g.renderTmp()
}

func (g *Generator) renderCmd() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, cmdDir, g.projectName), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderConfig() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, configDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderDeploy() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, deployDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderTmp() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, buildDir), defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(g.projectName, codegenDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderPkg() error {
	if err := os.MkdirAll(filepath.Join(g.projectName, apisDir, apiDirName(g.apiGroup), apiVersion(g.apiGroup)), defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(g.projectName, stubDir), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

// apiVersion extracts api version from the given apiGroup.
func apiVersion(apiGroup string) string {
	return strings.Split(apiGroup, "/")[1]
}

// groupName extracts the group name from the givem apiGroup.
func groupName(apiGroup string) string {
	return strings.Split(apiGroup, "/")[0]
}

// apiDirName extracts the name of api directory under ../apis/ from the apiGroup.
// it uses the first word separated with "." of the groupName as the api directory name.
// for example,
//  apiDirName("app.example.com/v1alpha1") => "app".
func apiDirName(apiGroup string) string {
	return strings.Split(groupName(apiGroup), ".")[0]
}
