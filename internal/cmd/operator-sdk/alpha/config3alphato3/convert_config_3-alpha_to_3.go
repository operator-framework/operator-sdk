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

package config3alphato3

import (
	"bufio"
	"bytes"
	"errors"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/mod/modfile"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/yaml"
)

var (
	v3alpha   = "3-alpha"
	versionRe = regexp.MustCompile(`version:[ ]*(?:")?3-alpha(?:")?`)
)

type templObj struct {
	IsGo      bool
	Resources []resource.Resource
}

// convertConfig3AlphaTo3 returns cfgBytes converted to 3 iff cfgBytes is version 3-alpha.
func convertConfig3AlphaTo3(cfgBytes []byte) (_ []byte, err error) {
	tObj := templObj{}
	cfgObj := make(map[string]interface{}, 5)
	if err := yaml.Unmarshal(cfgBytes, &cfgObj); err != nil {
		return nil, err
	}

	if version, hasVersion := cfgObj["version"]; !hasVersion || version.(string) != v3alpha {
		return cfgBytes, nil
	}

	cfgBytes = versionRe.ReplaceAll(cfgBytes, []byte(`version: "3"`))

	var modulePath string
	layout := cfgObj["layout"].(string)
	isGo := strings.HasPrefix(layout, "go.kubebuilder.io/")
	if isGo {
		if modulePath, err = getModulePath(); err != nil {
			return nil, err
		}
	}

	multigroupObj, hasMultigroup := cfgObj["multigroup"]
	isMultigroup := hasMultigroup && multigroupObj.(bool)

	if obj, hasRes := cfgObj["resources"]; hasRes {
		// Default to empty domain if no domain information is found.
		domain := ""
		if domainObj, ok := cfgObj["domain"]; ok {
			domain = domainObj.(string)
		}

		resObjs := obj.([]interface{})
		resources := make([]resource.Resource, len(resObjs))

		for i, resObj := range resObjs {
			resources[i].Domain = domain

			res := resObj.(map[string]interface{})
			if groupObj, ok := res["group"]; ok {
				resources[i].Group = groupObj.(string)
			}
			if versionObj, ok := res["version"]; ok {
				resources[i].Version = versionObj.(string)
			}
			if kindObj, ok := res["kind"]; ok {
				resources[i].Kind = kindObj.(string)
			}

			if isGo {
				// Only Go projects use "resources[*].path".
				var apiPath string
				if isMultigroup {
					apiPath = path.Join("apis", resources[i].Group, resources[i].Version)
				} else {
					apiPath = path.Join("api", resources[i].Version)
				}
				resources[i].Path = path.Join(modulePath, apiPath)
			}

			// Only set "resources[i].api" if "crdVersion" is present. Otherwise there is
			// likely no API defined by the project.
			if crdVersionObj, ok := res["crdVersion"]; ok {
				resources[i].API = &resource.API{
					CRDVersion: crdVersionObj.(string),
				}
			} else if k8sDomain, found := coreGroups[resources[i].Group]; found {
				// Core type case.
				resources[i].Domain = k8sDomain
				resources[i].Path = path.Join("k8s.io", "api", resources[i].Group, resources[i].Version)
			}
			// Only set "resources[i].webhooks" if "webhookVersion" is present. Otherwise there is
			// likely no webhook defined by the project.
			if whVersionObj, ok := res["webhookVersion"]; ok {
				resources[i].Webhooks = &resource.Webhooks{
					WebhookVersion: whVersionObj.(string),
				}
			}
		}

		tObj.Resources = resources
		tObj.IsGo = isGo

		out := bytes.Buffer{}
		t := template.Must(template.New("").Parse(tmpl))
		if err := t.Execute(&out, tObj); err != nil {
			return nil, err
		}

		// Scan for resources, then replace only reources to preserve comments/order of the rest of PROJECT.
		scanner := bufio.NewScanner(bytes.NewBuffer(cfgBytes))
		start, end := -1, len(cfgBytes)
		for scanner.Scan() {
			text := scanner.Text()
			if strings.HasPrefix(text, "resources:") {
				start = bytes.Index(cfgBytes, []byte("resources:"))
				continue
			}
			if start != -1 && strings.Contains(text, ":") && !strings.HasPrefix(text, " ") {
				key := strings.TrimRightFunc(text, func(c rune) bool {
					return c != ':'
				})
				if _, hasKey := cfgObj[strings.TrimSuffix(key, ":")]; hasKey {
					end = bytes.Index(cfgBytes, []byte(text))
					break
				}
			}
		}

		if start == -1 {
			return nil, errors.New("internal error: resources key not found in scanner")
		}
		cfgBytes = append(cfgBytes[:start], append(out.Bytes(), cfgBytes[end:]...)...)
	}

	return cfgBytes, nil
}

// Make this a var so it can be mocked in tests.
var getModulePath = func() (string, error) {
	b, err := ioutil.ReadFile("go.mod")
	return modfile.ModulePath(b), err
}

// Comment-heavy "resources" template.
const tmpl = `resources:
{{- $isGo := .IsGo }}
{{- range $i, $res := .Resources }}
-{{- if $res.API }} api:
    {{- if $res.API.CRDVersion }}
    crdVersion: {{ $res.API.CRDVersion }}
    {{- else }}
    # TODO(user): Change this API's CRD version if not v1.
    crdVersion: v1
    {{- end }}
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  {{- end }}
  {{- if $isGo }}
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  {{- end }}
  {{- if $res.Domain }}
  domain: {{ $res.Domain }}
  {{- end }}
  group: {{ $res.GVK.Group }}
  kind: {{ $res.GVK.Kind }}
  {{- if $res.Path }}
  # TODO(user): Update the package path for your API if the below value is incorrect.
  path: {{ $res.Path }}
  {{- end }}
  version: {{ $res.GVK.Version }}
  {{- if $res.Webhooks }}
  webhooks:
    # TODO(user): Uncomment the below line if this resource's webhook implements a conversion webhook, else delete it.
    # conversion: true
    # TODO(user): Uncomment the below line if this resource's webhook implements a defaulting webhook, else delete it.
    # defaulting: true
    # TODO(user): Uncomment the below line if this resource's webhook implements a validating webhook, else delete it.
    # validation: true
    {{- if $res.Webhooks.WebhookVersion }}
    webhookVersion: {{ $res.Webhooks.WebhookVersion }}
    {{- else }}
    # TODO(user): Change this API's webhook configuration version to the correct version if not v1.
    webhookVersion: v1
    {{- end }}
  {{- end }}
{{- end }}
`

// coreGroups maps a native k8s group to its domain. Several do not have a domain.
var coreGroups = map[string]string{
	"admission":             "k8s.io",
	"admissionregistration": "k8s.io",
	"apps":                  "",
	"auditregistration":     "k8s.io",
	"apiextensions":         "k8s.io",
	"authentication":        "k8s.io",
	"authorization":         "k8s.io",
	"autoscaling":           "",
	"batch":                 "",
	"certificates":          "k8s.io",
	"coordination":          "k8s.io",
	"core":                  "",
	"events":                "k8s.io",
	"extensions":            "",
	"imagepolicy":           "k8s.io",
	"networking":            "k8s.io",
	"node":                  "k8s.io",
	"metrics":               "k8s.io",
	"policy":                "",
	"rbac.authorization":    "k8s.io",
	"scheduling":            "k8s.io",
	"setting":               "k8s.io",
	"storage":               "k8s.io",
}
