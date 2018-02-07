package generator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/coreos/operator-sdk/pkg/templates"
)

const defaultFileMode = 0750

type Generator struct {
}

func (g *Generator) Render() error {
	gopath, ok := os.LookupEnv("GOPATH")
	if !ok {
		return errors.New("GOPATH must be set")
	}
	projDir, ok := os.LookupEnv("PROJECT") // github.com/coreos/play
	if !ok {
		return errors.New("PROJECT must be set")
	}
	apiGroup, ok := os.LookupEnv("APIGROUP") // play.coreos.com/v1alpha1
	if !ok {
		return errors.New("PROJECT must be set")
	}

	projDir = filepath.Join(gopath, "src", projDir)
	err := os.MkdirAll(projDir, defaultFileMode)
	if err != nil {
		return err
	}

	programName := func() string {
		splits := strings.Split(projDir, "/")
		return splits[len(splits)-1]
	}()
	err = os.MkdirAll(filepath.Join(projDir, "cmd", programName), defaultFileMode)
	if err != nil {
		return err
	}

	groupName, apiVersion := func() (string, string) {
		splits := strings.Split(apiGroup, "/")
		return strings.Split(splits[0], ".")[0], splits[1]
	}()
	err = os.MkdirAll(filepath.Join(projDir, "pkg/apis", groupName, apiVersion), defaultFileMode)
	if err != nil {
		return err
	}

	controllerDir := filepath.Join(projDir, "pkg/controller")
	err = os.MkdirAll(controllerDir, defaultFileMode)
	if err != nil {
		return err
	}

	err = g.genController(controllerDir, "controller.go", templates.ControllerTemplate)
	if err != nil {
		return err
	}
	err = g.genController(controllerDir, "workqueue.go", templates.WorkqueueTemplate)
	if err != nil {
		return err
	}

	return nil
}

func (g *Generator) genController(controllerDir, filename, tmpl string) error {
	f, err := os.OpenFile(filepath.Join(controllerDir, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	t, err := template.New(filename).Parse(tmpl)
	if err != nil {
		return err
	}
	err = t.Execute(f, nil)
	if err != nil {
		return err
	}

	return nil
}
