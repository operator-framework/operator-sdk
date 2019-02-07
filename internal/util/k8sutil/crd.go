package k8sutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/ghodss/yaml"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func GetCRDs(crdsDir string) ([]*apiextv1beta1.CustomResourceDefinition, error) {
	manifests, err := GetCRDManifestPaths(crdsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get CRD's from %s: (%v)", crdsDir, err)
	}
	crds := make([]*apiextv1beta1.CustomResourceDefinition, len(manifests))
	for i, m := range manifests {
		b, err := ioutil.ReadFile(m)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(b, crds[i]); err != nil {
			return nil, err
		}
	}
	return crds, nil
}

func GetCRDManifestPaths(crdsDir string) (crdPaths []string, err error) {
	err = filepath.Walk(crdsDir, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info == nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "_crd.yaml") {
			crdPaths = append(crdPaths, filepath.Join(crdsDir, info.Name()))
		}
		return nil
	})
	return crdPaths, err
}
