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

package olmcatalog

// InteractiveCSVCmd includes the list of CSV fields which would be asked from
// the user while CSV generation.
type InteractiveCSVcmd struct {
	// DisplayName is the name of the crd.
	DisplayName string
	// Keywords is a list of keywords describing the Operator.
	Keywords []string
	// Maintainers is a list of human or organizational entities maintaining the
	// Operator, with a name and email.
	Maintainers map[string]string
	// Provider is the name of the operator provider with a name.
	Provider map[string]string
	// List of key, value pairs which can be added and are used by Operator internals.
	Labels map[string]string
	// A minimum version of Kubernetes that server is supposed to have so operator(s)
	// can be deployed. The Kubernetes version must be in "Major.Minor.Patch"
	// format (e.g: 1.11.0).
	MinKubeVersion string
	// A list of relevant links for the Operator. Common links include documentation,
	// how-to guides, blog posts, and the company homepage
	Links map[string]string
}
