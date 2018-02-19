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
	clientDir       = pkgDir + "/client"
	stubDir         = pkgDir + "/stub"
)

type Generator struct {
	apiGroup    string
	kind        string
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
	if err := g.renderTmp(); err != nil {
		return err
	}
	return nil
}

func (g *Generator) renderCmd() error {
	if err := os.MkdirAll(filepath.Join(cmdDir, g.projectName), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderConfig() error {
	if err := os.MkdirAll(configDir, defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderDeploy() error {
	if err := os.MkdirAll(filepath.Join(deployDir, g.projectName), defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderTmp() error {
	if err := os.MkdirAll(buildDir, defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(codegenDir, defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

func (g *Generator) renderPkg() error {
	if err := os.MkdirAll(filepath.Join(apisDir, g.projectName, apiVersion(g.apiGroup)), defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(clientDir, defaultFileMode); err != nil {
		return err
	}
	if err := os.MkdirAll(stubDir, defaultFileMode); err != nil {
		return err
	}
	// TODO render files.
	return nil
}

// apiVersion extracts api version from the given apiGroup.
func apiVersion(apiGroup string) string {
	return strings.Split(apiGroup, "/")[1]
}
