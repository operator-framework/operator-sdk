package scaffold

import (
	"log"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const (
	// test constants describing an app operator project
	appProjectName = "app-operator"
	appRepo        = "github.com" + filePathSep + "example-inc" + filePathSep + appProjectName
	appApiVersion  = "app.example.com/v1alpha1"
	appKind        = "AppService"
)

func mustGetImportPath() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("mustGetImportPath: ", err)
	}
	return filepath.Join(wd, appRepo)
}

var (
	appConfig = &input.Config{
		Repo:        appRepo,
		ProjectPath: mustGetImportPath(),
		ProjectName: appProjectName,
	}
)
