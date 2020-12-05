module github.com/operator-framework/operator-sdk

go 1.15

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/fatih/structtag v1.1.0
	github.com/go-logr/logr v0.3.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/kr/text v0.1.0
	github.com/markbates/inflect v1.0.4
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/operator-framework/api v0.3.20
	github.com/operator-framework/operator-lib v0.2.1-0.20201119204525-6700ae54da56
	github.com/operator-framework/operator-registry v1.14.3
	github.com/prometheus/client_golang v1.7.1
	github.com/sergi/go-diff v1.0.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/tools v0.0.0-20201014231627-1610a49f37af
	gomodules.xyz/jsonpatch/v3 v3.0.1
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.19.4
	k8s.io/apiextensions-apiserver v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/cli-runtime v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/kubectl v0.19.4
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.7.0-alpha.7
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/kubebuilder/v2 v2.3.2-0.20201204230216-593f2e80cdf6
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
