// Copyright 2021 The Operator-SDK Authors
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
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	StorageSidecarContainer = "scorecard-gather"
)

func (r PodTestRunner) execInPod(podName, mountPath, containerName string) (io.Reader, io.Reader, error) {
	cmd := []string{
		"tar",
		"cf",
		"-",
		mountPath,
	}

	stdoutReader, outStream := io.Pipe()
	stderrReader, errStream := io.Pipe()
	const tty = false
	req := r.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(r.Namespace).SubResource("exec").Param("container", containerName)
	req.VersionedParams(
		&v1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     tty,
		},
		scheme.ParameterCodec,
	)

	exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", req.URL())
	if err != nil {
		return nil, nil, err
	}

	go func() {
		defer outStream.Close()
		defer errStream.Close()
		err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: outStream,
			Stderr: errStream,
		})
		if err != nil {
			log.Error(err)
		}
	}()
	return stdoutReader, stderrReader, err
}

func getStoragePrefix(file string) string {
	return strings.TrimLeft(file, "/")
}

func untarAll(reader io.Reader, destDir, prefix string) error {
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if !strings.HasPrefix(header.Name, prefix) {
			return fmt.Errorf("tar contents corrupted")
		}

		mode := header.FileInfo().Mode()
		destFileName := filepath.Join(destDir, header.Name[len(prefix):])

		baseName := filepath.Dir(destFileName)
		if err := os.MkdirAll(baseName, 0755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(destFileName, 0755); err != nil {
				return err
			}
			continue
		}

		if mode&os.ModeSymlink != 0 {
			linkname := header.Linkname

			if err := os.Symlink(linkname, destFileName); err != nil {
				return err
			}
		} else {
			outFile, err := os.Create(destFileName)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

func addStorageToPod(podDef *v1.Pod, mountPath string, storageImage string) {

	// add the emptyDir volume for storage to the test Pod
	newVolume := v1.Volume{}
	newVolume.Name = "scorecard-storage"
	newVolume.VolumeSource = v1.VolumeSource{}

	podDef.Spec.Volumes = append(podDef.Spec.Volumes, newVolume)

	// add the storage sidecar container
	storageContainer := v1.Container{
		Name:            StorageSidecarContainer,
		Image:           storageImage,
		ImagePullPolicy: v1.PullIfNotPresent,
		Args: []string{
			"/bin/sh",
			"-c",
			//"trap 'echo TERM;exit 0' TERM;tail -f /dev/null",
			"sleep 1000",
		},
		VolumeMounts: []v1.VolumeMount{
			{
				MountPath: mountPath,
				Name:      "scorecard-storage",
				ReadOnly:  true,
			},
		},
	}

	podDef.Spec.Containers = append(podDef.Spec.Containers, storageContainer)

	// add the storage emptyDir volume into the test container

	vMount := v1.VolumeMount{
		MountPath: mountPath,
		Name:      "scorecard-storage",
		ReadOnly:  false,
	}
	podDef.Spec.Containers[0].VolumeMounts = append(podDef.Spec.Containers[0].VolumeMounts, vMount)

	// add mountPath to Env
	mountPathEnv := v1.EnvVar{
		Name:  "SCORECARD_STORAGE",
		Value: mountPath,
	}
	podDef.Spec.Containers[0].Env = append(podDef.Spec.Containers[0].Env, mountPathEnv)

}

func gatherTestOutput(r PodTestRunner, suiteName, testName, podName, mountPath string) error {

	//exec into sidecar container, run tar,  get reader
	containerName := StorageSidecarContainer
	stdoutReader, stderrReader, err := r.execInPod(podName, mountPath, containerName)
	if err != nil {
		return err
	}

	srcPath := mountPath
	prefix := getStoragePrefix(srcPath)
	prefix = path.Clean(prefix)
	destPath := getDestPath(r.TestOutput, suiteName, testName)
	err = untarAll(stdoutReader, destPath, prefix)
	if err != nil {
		return err
	}
	stderr, err := io.ReadAll(stderrReader)
	if err != nil {
		return err
	}
	if len(stderr) > 0 {
		destFileName := filepath.Join(destPath, "tar_stderr")
		err = os.WriteFile(destFileName, stderr, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func getDestPath(baseDir, suiteName, testName string) (destPath string) {
	destPath = baseDir + string(os.PathSeparator)
	if suiteName != "" {
		destPath = destPath + suiteName + string(os.PathSeparator)
	}
	destPath = destPath + testName
	return destPath
}
