package main

import (
	ansibleImage "github.com/operator-framework/operator-sdk/pkg/ansible/image"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	log "github.com/sirupsen/logrus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func main() {
	logf.SetLogger(zap.Logger())

	err := ansibleImage.RunAnsibleOperator()
	if err != nil {
		log.Error(err, "error running ansible operator binary")
	}
}
