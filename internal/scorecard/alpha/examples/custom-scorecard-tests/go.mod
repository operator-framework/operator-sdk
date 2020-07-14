module github.com/username/custom-scorecard-tests

go 1.13

replace k8s.io/client-go => k8s.io/client-go v0.18.2

require (
	github.com/operator-framework/api v0.3.8
	github.com/operator-framework/operator-registry v1.12.6-0.20200611222234-275301b779f8
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/sirupsen/logrus v1.5.0
)
