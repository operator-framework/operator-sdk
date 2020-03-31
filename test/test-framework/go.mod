module github.com/operator-framework/operator-sdk/test/test-framework

go 1.13

require (
	github.com/operator-framework/operator-sdk v0.0.0
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d // Required by Helm
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48 // Required by prometheus-operator
)

replace github.com/operator-framework/operator-sdk => ../../
