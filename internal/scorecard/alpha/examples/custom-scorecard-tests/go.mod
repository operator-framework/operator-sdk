module github.com/username/custom-scorecard-tests

go 1.13

replace k8s.io/client-go => k8s.io/client-go v0.18.2

require (
	github.com/jmccormick2001/custom-scorecard-tests v0.0.0-20200515184210-f02f013bb1a7
	github.com/operator-framework/api v0.3.4
	github.com/operator-framework/operator-registry v1.12.3
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/sirupsen/logrus v1.5.0
)
