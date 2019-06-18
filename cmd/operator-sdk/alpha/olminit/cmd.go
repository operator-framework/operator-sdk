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

package olminit

import (
	"context"
	"time"

	"github.com/operator-framework/operator-sdk/internal/olm"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	version string
	timeout time.Duration
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "olm-init",
		Short: "Initialize Operator Lifecycle manager in your cluster",
		RunE:  initOLM,
	}

	cmd.Flags().StringVar(&version, "version", "latest", "Version of OLM to initialize")
	cmd.Flags().DurationVar(&timeout, "timeout", time.Second*60, "Timeout duration to wait for OLM for become ready before outputting status")
	return cmd
}

func initOLM(cmd *cobra.Command, args []string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Failed to get kubernetes config: %s", err)
	}

	olm, err := olm.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create OLM initializer: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	status, err := olm.InstallVersion(ctx, version)
	if err != nil {
		log.Fatalf("Failed to initialize OLM: %s", err)
	}

	log.Info(*status)
	return nil
}
