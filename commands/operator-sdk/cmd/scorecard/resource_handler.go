// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scorecard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	proxyConf "github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func createFromYAMLFile(yamlPath string, storeKindVersionName bool) error {
	yamlFile, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", yamlPath, err)
	}
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		obj := &unstructured.Unstructured{}
		jsonSpec, err := yaml.YAMLToJSON(yamlSpec)
		if err != nil {
			return fmt.Errorf("could not convert yaml file to json: %v", err)
		}
		obj.UnmarshalJSON(jsonSpec)
		obj.SetNamespace(SCConf.Namespace)
		if storeKindVersionName {
			apiversion = obj.GetAPIVersion()
			kind = obj.GetKind()
			name = obj.GetName()
		}

		// dirty hack to merge scorecard proxy into operator deployment; lots of serialization and deserialization
		if obj.GetKind() == "Deployment" {
			// TODO: support multiple deployments
			if deploymentName != "" {
				return fmt.Errorf("scorecard currently does not support multiple deployments in the manifests")
			}
			dep, err := unstructuredToDeployment(obj)
			if err != nil {
				return fmt.Errorf("failed to convert object to deployment: %v", err)
			}
			deploymentName = dep.GetName()
			createKubeconfigSecret()
			addMountKubeconfigSecret(dep)
			addProxyContainer(dep)
			// go back to unstructured to create
			obj, err = deploymentToUnstructured(dep)
			if err != nil {
				return fmt.Errorf("failed to convert deployment to unstructured: %v", err)
			}
		}
		err = runtimeClient.Create(context.TODO(), obj)
		if err != nil {
			// not sure if := will copy just a pointer or do a full copy, so for now just use fmt.Errorf
			oldErr := fmt.Errorf("%v", err)
			_, err := restMapper.RESTMappings(obj.GetObjectKind().GroupVersionKind().GroupKind())
			if err == nil {
				return oldErr
			}
			// don't store error, as only error will be timeout. Error from runtime client will be easier for
			// the user to understand than the timeout error, so just use that if we fail
			wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {
				restMapper.Reset()
				_, err := restMapper.RESTMappings(obj.GetObjectKind().GroupVersionKind().GroupKind())
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			err = runtimeClient.Create(context.TODO(), obj)
			if err != nil {
				return err
			}
		}
		addResourceCleanup(obj, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
	}
	return nil
}

func createKubeconfigSecret() {
	kubeconfigMap := make(map[string][]byte)
	kc, err := proxyConf.Create(metav1.OwnerReference{Name: "scorecard"}, "http://localhost:8888", SCConf.Namespace)
	defer os.Remove(kc.Name())
	kc, err = os.Open(kc.Name())
	if err != nil {
		log.Fatal(err)
	}
	kcBytes, err := ioutil.ReadAll(kc)
	if err != nil {
		log.Fatal(err)
	}
	kubeconfigMap["kubeconfig"] = kcBytes
	kubeconfigSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scorecard-kubeconfig",
			Namespace: SCConf.Namespace,
		},
		Data: kubeconfigMap,
	}
	runtimeClient.Create(context.TODO(), kubeconfigSecret)
}

func addMountKubeconfigSecret(dep *appsv1.Deployment) {
	// create mount for secret
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, v1.Volume{
		Name: "scorecard-kubeconfig",
		VolumeSource: v1.VolumeSource{Secret: &v1.SecretVolumeSource{
			SecretName: "scorecard-kubeconfig",
			Items: []v1.KeyToPath{{
				Key:  "kubeconfig",
				Path: "config",
			}},
		},
		},
	})
	for index := range dep.Spec.Template.Spec.Containers {
		// mount the volume
		dep.Spec.Template.Spec.Containers[index].VolumeMounts = append(dep.Spec.Template.Spec.Containers[index].VolumeMounts, v1.VolumeMount{
			Name:      "scorecard-kubeconfig",
			MountPath: "/scorecard-secret",
		})
		// specify the path via KUBECONFIG env var
		dep.Spec.Template.Spec.Containers[index].Env = append(dep.Spec.Template.Spec.Containers[index].Env, v1.EnvVar{
			Name:  "KUBECONFIG",
			Value: "/scorecard-secret/config",
		})
	}
}

func addProxyContainer(dep *appsv1.Deployment) {
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, v1.Container{
		Name:            "scorecard-proxy",
		Image:           "scorecard-proxy",
		ImagePullPolicy: "Never",
		Command:         []string{"scorecard-proxy"},
		Env: []v1.EnvVar{{
			Name:      k8sutil.WatchNamespaceEnvVar,
			ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
		}},
	})
}

func unstructuredToDeployment(obj *unstructured.Unstructured) (*appsv1.Deployment, error) {
	jsonByte, err := obj.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to convert deployment to json: %v", err)
	}
	depObj, _, err := dynamicDecoder.Decode(jsonByte, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode deployment object: %v", err)
	}
	switch o := depObj.(type) {
	case *appsv1.Deployment:
		return o, nil
	default:
		return nil, fmt.Errorf("conversion of runtime object to deployment failed (resulting runtime object not deployment type)")
	}
}

func deploymentToUnstructured(dep *appsv1.Deployment) (*unstructured.Unstructured, error) {
	jsonByte, err := json.Marshal(dep)
	if err != nil {
		return nil, fmt.Errorf("failed to remarshal deployment: %v", err)
	}
	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(jsonByte)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated deployment: %v", err)
	}
	return obj, nil
}

func cleanupScorecard() error {
	failed := false
	for i := len(cleanupFns) - 1; i >= 0; i-- {
		err := cleanupFns[i]()
		if err != nil {
			failed = true
			log.Printf("a cleanup function failed with error: %v\n", err)
		}
	}
	if failed {
		return fmt.Errorf("a cleanup function failed; see stdout for more details")
	}
	return nil
}

func addResourceCleanup(obj runtime.Object, key types.NamespacedName) {
	cleanupFns = append(cleanupFns, func() error {
		// make a copy of the object because the client changes it
		objCopy := obj.DeepCopyObject()
		err := runtimeClient.Delete(context.TODO(), obj)
		if err != nil {
			return err
		}
		err = wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {
			err = runtimeClient.Get(context.TODO(), key, objCopy)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, fmt.Errorf("error encountered during deletion of resource type %v with namespace/name (%+v): %v", objCopy.GetObjectKind().GroupVersionKind().Kind, key, err)
			}
			return false, nil
		})
		if err != nil {
			return fmt.Errorf("cleanup function failed: %v", err)
		}
		return nil
	})
}
