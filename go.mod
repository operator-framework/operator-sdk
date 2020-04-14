module github.com/operator-framework/operator-sdk

go 1.13

require (
	github.com/DATA-DOG/go-sqlmock v1.4.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/prometheus-operator v0.38.0
	github.com/fatih/structtag v1.1.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/gobuffalo/packr v1.30.1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/helm/helm-2to3 v0.5.1
	github.com/iancoleman/strcase v0.0.0-20190422225806-e506e3ef7365
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/markbates/inflect v1.0.4
	github.com/martinlindhe/base36 v1.0.0
	github.com/mattn/go-isatty v0.0.12
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/operator-framework/api v0.1.1
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20200321030439-57b580e57e88
	github.com/operator-framework/operator-registry v1.6.2-0.20200330184612-11867930adb5
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/rogpeppe/go-internal v1.5.0
	github.com/rubenv/sql-migrate v0.0.0-20191025130928-9355dd04f4b3 // indirect
	github.com/sergi/go-diff v1.0.0
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/ziutek/mymysql v1.5.4 // indirect
	go.uber.org/zap v1.14.1
	golang.org/x/tools v0.0.0-20200327195553-82bb89366a1e
	gomodules.xyz/jsonpatch/v3 v3.0.1
	gopkg.in/gorp.v1 v1.7.2 // indirect
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.1.2
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/cli-runtime v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.17.4
	k8s.io/gengo v0.0.0-20191010091904-7fa3014cb28f
	k8s.io/helm v2.16.3+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-state-metrics v1.7.2
	k8s.io/kubectl v0.17.4
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/controller-tools v0.2.8
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
