package common

import (
	"fmt"
	"os"
)

var saSecretTemplate = `---
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  name: %s
  annotations:
    kubernetes.io/service-account.name: "%s"
`

// GetSASecret writes a service account token secret to a file. It returns a string to the file or an error if it fails to write the file
func GetSASecret(name string, dir string) (string, error) {
	secretName := name + "-secret"
	fileName := dir + "/" + secretName + ".yaml"
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(saSecretTemplate, secretName, name)), 0777)
	if err != nil {
		return "", err
	}

	return fileName, nil
}
