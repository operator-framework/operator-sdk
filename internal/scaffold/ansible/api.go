// Copyright 2020 The Operator-SDK Authors
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

package ansible

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/flags/apiflags"
	"github.com/operator-framework/operator-sdk/internal/genutil"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

//

// TODO
// Consolidate scaffold func to be used by both "new" and "add api" commands.
func API(cfg input.Config, apiFlags apiflags.APIFlags) error {
	// Create and validate new resource.
	r, err := scaffold.NewResource(apiFlags.APIVersion, apiFlags.Kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: %v", err)
	}
	roleFiles := RolesFiles{Resource: *r}
	roleTemplates := RolesTemplates{Resource: *r}

	// update watch.yaml for the given resource r.
	if err := UpdateAnsibleWatchForResource(r, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the Watch manifest for the resource (%v, %v): (%v)",
			r.APIVersion, r.Kind, err)
	}

	s := &scaffold.Scaffold{}
	err = s.Execute(&cfg,
		&scaffold.CR{Resource: r},
		&RolesReadme{Resource: *r},
		&RolesMetaMain{Resource: *r},
		&roleFiles,
		&roleTemplates,
		&RolesVarsMain{Resource: *r},
		&RolesDefaultsMain{Resource: *r},
		&RolesTasksMain{Resource: *r},
		&RolesHandlersMain{Resource: *r},
	)
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}
	if err = genutil.GenerateCRDNonGo("", *r, apiFlags.CrdVersion); err != nil {
		return err
	}

	// Remove placeholders from empty directories
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleFiles.Path))
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleTemplates.Path))
	if err != nil {
		return fmt.Errorf("new ansible api scaffold failed: %v", err)
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(r, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): (%v)",
			r.APIVersion, r.Kind, err)
	}
	return nil
}
