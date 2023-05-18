// Copyright 2023 The Operator-SDK Authors
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

package fbcindex

import (
	"bytes"
	"compress/gzip"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	yamlSeparator    = "\n---\n"
	gzipSuffixLength = 13
	maxGZIPLength    = maxConfigMapSize - gzipSuffixLength

	ConfigMapEncodingAnnotationKey  = "olm.contentEncoding"
	ConfigMapEncodingAnnotationGzip = "gzip+base64"
)

/*
This file implements the actual building of the CM list. It uses the template method design pattern to implement both
regular string VM, and compressed binary CM.

The method itself is FBCRegistryPod.getConfigMaps. This file contains the actual implementation of the writing actions,
used by the method.
*/

type configMapWriter interface {
	reset()
	newConfigMap(string) *corev1.ConfigMap
	getFilePath() string
	isEmpty() bool
	exceedMaxLength(cmSize int, data string) (bool, error)
	closeCM(cm *corev1.ConfigMap) error
	addData(data string) error
	continueAddData(data string) error
	writeLastFragment(cm *corev1.ConfigMap) error
}

type gzipCMWriter struct {
	actualBuff   *bytes.Buffer
	helperBuff   *bytes.Buffer
	actualWriter *gzip.Writer
	helperWriter *gzip.Writer
	cmName       string
	namespace    string
}

func newGZIPWriter(name, namespace string) *gzipCMWriter {
	actualBuff := &bytes.Buffer{}
	helperBuff := &bytes.Buffer{}

	return &gzipCMWriter{
		actualBuff:   actualBuff,
		helperBuff:   helperBuff,
		actualWriter: gzip.NewWriter(actualBuff),
		helperWriter: gzip.NewWriter(helperBuff),
		cmName:       name,
		namespace:    namespace,
	}
}

func (cmw *gzipCMWriter) reset() {
	cmw.actualBuff.Reset()
	cmw.actualWriter.Reset(cmw.actualBuff)
	cmw.helperBuff.Reset()
	cmw.helperWriter.Reset(cmw.helperBuff)
}

func (cmw *gzipCMWriter) newConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cmw.namespace,
			Name:      name,
			Annotations: map[string]string{
				ConfigMapEncodingAnnotationKey: ConfigMapEncodingAnnotationGzip,
			},
		},
		BinaryData: map[string][]byte{},
	}
}

func (cmw *gzipCMWriter) getFilePath() string {
	return fmt.Sprintf("%s.yaml.gz", defaultConfigMapKey)
}

func (cmw *gzipCMWriter) isEmpty() bool {
	return cmw.actualBuff.Len() > 0
}

func (cmw *gzipCMWriter) exceedMaxLength(cmSize int, data string) (bool, error) {
	_, err := cmw.helperWriter.Write([]byte(data))
	if err != nil {
		return false, err
	}

	err = cmw.helperWriter.Flush()
	if err != nil {
		return false, err
	}

	return cmSize+cmw.helperBuff.Len() > maxGZIPLength, nil
}

func (cmw *gzipCMWriter) closeCM(cm *corev1.ConfigMap) error {
	err := cmw.actualWriter.Close()
	if err != nil {
		return err
	}

	err = cmw.actualWriter.Flush()
	if err != nil {
		return err
	}

	cm.BinaryData[defaultConfigMapKey] = make([]byte, cmw.actualBuff.Len())
	copy(cm.BinaryData[defaultConfigMapKey], cmw.actualBuff.Bytes())

	cmw.reset()

	return nil
}

func (cmw *gzipCMWriter) addData(data string) error {
	dataBytes := []byte(data)
	_, err := cmw.helperWriter.Write(dataBytes)
	if err != nil {
		return err
	}
	_, err = cmw.actualWriter.Write(dataBytes)
	if err != nil {
		return err
	}
	return nil
}

// continueAddData completes adding the data after starting adding it in exceedMaxLength
func (cmw *gzipCMWriter) continueAddData(data string) error {
	_, err := cmw.actualWriter.Write([]byte(data))
	if err != nil {
		return err
	}
	return nil
}

func (cmw *gzipCMWriter) writeLastFragment(cm *corev1.ConfigMap) error {
	err := cmw.actualWriter.Close()
	if err != nil {
		return err
	}

	cm.BinaryData[defaultConfigMapKey] = cmw.actualBuff.Bytes()
	return nil
}
