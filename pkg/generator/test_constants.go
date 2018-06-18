package generator

const (
	// test constants for app-operator
	appImage       = "quay.io/example-inc/app-operator:0.0.1"
	appRepoPath    = "github.com/example-inc/" + appProjectName
	appKind        = "AppService"
	appApiDirName  = "app"
	appAPIVersion  = appGroupName + "/" + appVersion
	appVersion     = "v1alpha1"
	appGroupName   = "app.example.com"
	appProjectName = "app-operator"
	errorMessage   = "Want:\n%v\nGot:\n%v"
)
