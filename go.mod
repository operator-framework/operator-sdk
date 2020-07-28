module github.com/operator-framework/operator-sdk

go 1.13

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/fatih/structtag v1.1.0
	github.com/go-logr/logr v0.1.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/kr/text v0.1.0
	github.com/kubernetes-sigs/kustomize v2.0.3+incompatible // indirect
	github.com/markbates/inflect v1.0.4
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/api v0.3.8
	github.com/operator-framework/operator-lib v0.1.0
	github.com/operator-framework/operator-registry v1.13.4
	github.com/prometheus/client_golang v1.5.1
	github.com/sergi/go-diff v1.0.0
	github.com/sirupsen/logrus v1.5.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/tools v0.0.0-20200403190813-44a64ad78b9b
	gomodules.xyz/jsonpatch/v3 v3.0.1
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.4
	k8s.io/apiextensions-apiserver v0.18.4
	k8s.io/apimachinery v0.18.4
	k8s.io/cli-runtime v0.18.2
	k8s.io/client-go v0.18.4
	k8s.io/kubectl v0.18.2
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.1
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/kubebuilder v1.0.9-0.20200724202016-21f9343e992e
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
)

// This replaced version includes controller-runtime predicate utilities necessary for v1.0.0 that are still in master.
// Remove this and require the next minor/patch version of controller-runtime (>v0.6.1) when released.
replace sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.1-0.20200724132623-e50c7b819263
