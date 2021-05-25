module github.com/operator-framework/operator-sdk

go 1.16

require (
	github.com/blang/semver/v4 v4.0.0
	github.com/fatih/structtag v1.1.0
	github.com/go-logr/logr v0.3.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/kr/text v0.1.0
	github.com/markbates/inflect v1.0.4
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.2
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/operator-framework/api v0.8.1
	github.com/operator-framework/java-operator-plugins v0.0.0-20210525141944-8303c38a876d
	github.com/operator-framework/operator-lib v0.4.1
	github.com/operator-framework/operator-registry v1.15.3
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/client_model v0.2.0
	github.com/sergi/go-diff v1.1.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/thoas/go-funk v0.8.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.4.2
	golang.org/x/sys v0.0.0-20210521090106-6ca3eb03dfc2 // indirect
	golang.org/x/tools v0.1.1
	gomodules.xyz/jsonpatch/v3 v3.0.1
	helm.sh/helm/v3 v3.4.1
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/cli-runtime v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/kubectl v0.20.2
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.5.0
	sigs.k8s.io/kubebuilder/v3 v3.0.0-beta.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	// Using containerd 1.4.0+ resolves an issue with invalid error logging
	// from an init function in containerd. This replace can be removed when
	// one of our direct dependencies begins using containerd v1.4.0+
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.3
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
	sigs.k8s.io/kubebuilder/v3 => sigs.k8s.io/kubebuilder/v3 v3.0.0-alpha.0.0.20210518234629-191170994550
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
