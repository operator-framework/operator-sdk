package alpha

import (
	"fmt"
	"io/ioutil"
	"os"
)

func getBundleData(bundlePath string) (bundleData []byte, err error) {

	// make sure the bundle exists on disk
	_, err = os.Stat(bundlePath)
	if os.IsNotExist(err) {
		return bundleData, fmt.Errorf("bundle path is not valid %s", err.Error())
	}

	paths := []string{bundlePath}
	err = Tartar("/tmp/my.tar", paths)
	if err != nil {
		return bundleData, fmt.Errorf("error creating tar of bundle %s", err.Error())
	}

	var buf []byte
	buf, err = ioutil.ReadFile("/tmp/my.tar")
	if err != nil {
		return bundleData, fmt.Errorf("error reading tar of bundle %s", err.Error())
	}

	fmt.Printf("bundle bytes len %d\n", len(buf))

	return buf, err
}
