// Copyright 2019 The Operator-SDK Authors
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

package olm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	DefaultVersion = "latest"
	DefaultTimeout = time.Minute * 2
)

type Manager struct {
	Client  *Client
	Version string
	Timeout time.Duration

	once sync.Once
}

func (m *Manager) initialize() (err error) {
	m.once.Do(func() {
		if m.Client == nil {
			cfg, err := config.GetConfig()
			if err != nil {
				err = errors.Wrapf(err, "failed to get Kubernetes config")
				return
			}

			client, err := ClientForConfig(cfg)
			if err != nil {
				err = errors.Wrapf(err, "failed to create manager client")
				return
			}
			m.Client = client
		}
		if m.Version == "" {
			m.Version = DefaultVersion
		}
		if m.Timeout <= 0 {
			m.Timeout = DefaultTimeout
		}
	})
	return err
}

func (m *Manager) Install() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	status, err := m.Client.InstallVersion(ctx, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully installed OLM version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}

func (m *Manager) Uninstall() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	if err := m.Client.UninstallVersion(ctx, m.Version); err != nil {
		return err
	}

	log.Infof("Successfully uninstalled OLM version %q", m.Version)
	return nil
}

func (m *Manager) Status() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	status, err := m.Client.GetStatus(ctx, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully got OLM status for version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}

func (m *Manager) AddToFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.Version, "version", DefaultVersion, "version of OLM resources to install, uninstall, or get status about")
	fs.DurationVar(&m.Timeout, "timeout", DefaultTimeout, "time to wait for the command to complete before failing")
}
