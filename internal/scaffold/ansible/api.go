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
