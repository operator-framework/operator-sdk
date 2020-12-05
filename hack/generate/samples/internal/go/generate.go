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

package pkg

import (
	"path/filepath"

	golangv2 "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go/v2"
	golangv3 "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go/v3"
)

func GenerateMemcachedGoWithWebhooksSample(rootPath string) {
	golangv2.GenerateMemcachedGoWithWebhooksSample(filepath.Join(rootPath, "go", "v2"))
	golangv3.GenerateMemcachedGoWithWebhooksSample(filepath.Join(rootPath, "go", "v3"))
}
