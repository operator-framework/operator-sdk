package ansible

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/internal/flags/apiflags"
	"github.com/operator-framework/operator-sdk/internal/genutil"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

type InitOptions struct {
	GeneratePlaybook   bool
	ResourceAPIVersion string
	ResourceKind       string
	CRDVersion         string
}

func Init(cfg input.Config, generatePlaybook bool, apiFlags apiflags.APIFlags) error {
	resource, err := scaffold.NewResource(apiFlags.APIVersion, apiFlags.Kind)
	if err != nil {
		return fmt.Errorf("invalid apiVersion and kind: %v", err)
	}

	roleFiles := RolesFiles{Resource: *resource}
	roleTemplates := RolesTemplates{Resource: *resource}

	s := &scaffold.Scaffold{}
	err = s.Execute(&cfg,
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.CR{Resource: resource},
		&BuildDockerfile{GeneratePlaybook: generatePlaybook},
		&RolesReadme{Resource: *resource},
		&RolesMetaMain{Resource: *resource},
		&roleFiles,
		&roleTemplates,
		&RolesVarsMain{Resource: *resource},
		&MoleculeTestLocalConverge{Resource: *resource},
		&RolesDefaultsMain{Resource: *resource},
		&RolesTasksMain{Resource: *resource},
		&MoleculeDefaultMolecule{},
		&MoleculeDefaultPrepare{},
		&MoleculeDefaultConverge{
			GeneratePlaybook: generatePlaybook,
			Resource:         *resource,
		},
		&MoleculeDefaultVerify{},
		&RolesHandlersMain{Resource: *resource},
		&Watches{
			GeneratePlaybook: generatePlaybook,
			Resource:         *resource,
		},
		&DeployOperator{},
		&Travis{},
		&RequirementsYml{},
		&MoleculeTestLocalMolecule{},
		&MoleculeTestLocalPrepare{},
		&MoleculeTestLocalVerify{},
		&MoleculeClusterMolecule{Resource: *resource},
		&MoleculeClusterCreate{},
		&MoleculeClusterPrepare{Resource: *resource},
		&MoleculeClusterConverge{},
		&MoleculeClusterVerify{Resource: *resource},
		&MoleculeClusterDestroy{Resource: *resource},
		&MoleculeTemplatesOperator{},
	)
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}

	if err = genutil.GenerateCRDNonGo("", *resource, apiFlags.CrdVersion); err != nil {
		return err
	}

	// // Remove placeholders from empty directories
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleFiles.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}
	err = os.Remove(filepath.Join(s.AbsProjectPath, roleTemplates.Path))
	if err != nil {
		return fmt.Errorf("new ansible scaffold failed: %v", err)
	}

	if generatePlaybook {
		log.Info("Generating Ansible playbook.")

		err := s.Execute(&cfg,
			&Playbook{Resource: *resource},
		)
		if err != nil {
			return fmt.Errorf("new ansible playbook scaffold failed: %v", err)
		}
	}

	// update deploy/role.yaml for the given resource r.
	if err := scaffold.UpdateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		return fmt.Errorf("failed to update the RBAC manifest for the resource (%v, %v): %v",
			resource.APIVersion, resource.Kind, err)
	}
	return nil
}
