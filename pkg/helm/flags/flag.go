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

package flags

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/internal/flags"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"

	"github.com/spf13/pflag"
	"k8s.io/helm/pkg/storage/driver"
)

// HelmOperatorFlags - Options to be used by a helm operator
type HelmOperatorFlags struct {
	flags.WatchFlags
	storageFlags
}

// AddTo - Add the helm operator flags to the flagset.
// helpTextPrefix will allow you add a prefix to default help text. Joined by a space.
func AddTo(flagSet *pflag.FlagSet, helpTextPrefix ...string) *HelmOperatorFlags {
	hof := &HelmOperatorFlags{}
	hof.WatchFlags.AddTo(flagSet, helpTextPrefix...)
	hof.storageFlags.AddTo(flagSet, helpTextPrefix...)
	flagSet.AddFlagSet(zap.FlagSet())
	return hof
}

type storageFlags struct {
	StorageDriver    string
	StorageNamespace string
}

// AddTo - Add the helm operator storage flags to the flagset.
// helpTextPrefix will allow you add a prefix to default help text. Joined by a space.
func (sf *storageFlags) AddTo(flagSet *pflag.FlagSet, helpTextPrefix ...string) {
	driverNames := fmt.Sprintf("'%s', '%s', or '%s'", driver.ConfigMapsDriverName, driver.SecretsDriverName, driver.MemoryDriverName)
	flagSet.StringVar(&sf.StorageDriver,
		"storage-driver",
		driver.ConfigMapsDriverName,
		strings.Join(append(helpTextPrefix, "Storage driver to use. One of "+driverNames), " "),
	)
	flagSet.StringVar(&sf.StorageNamespace,
		"storage-namespace",
		"",
		strings.Join(append(helpTextPrefix, "Namespace used by the storage driver for persisting release information"), " "),
	)
}
