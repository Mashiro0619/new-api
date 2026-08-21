/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package site_model_calls

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const OptionKey = "AllSiteModelCalls"

const DefaultSerialized = `{"enabled":false,"models":[]}`

type Config struct {
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models"`
}

func DefaultConfig() Config {
	return Config{Enabled: false, Models: []string{}}
}

func Normalize(config Config) Config {
	seen := make(map[string]struct{}, len(config.Models))
	models := make([]string, 0, len(config.Models))
	for _, raw := range config.Models {
		modelName := strings.TrimSpace(raw)
		if modelName == "" {
			continue
		}
		if _, exists := seen[modelName]; exists {
			continue
		}
		seen[modelName] = struct{}{}
		models = append(models, modelName)
	}
	sort.Strings(models)
	config.Models = models
	return config
}

func Parse(raw string) (Config, error) {
	if strings.TrimSpace(raw) == "" {
		return DefaultConfig(), nil
	}

	var object map[string]any
	if err := common.UnmarshalJsonStr(raw, &object); err != nil {
		return DefaultConfig(), err
	}
	if object == nil {
		return DefaultConfig(), errors.New("site model calls config must be a JSON object")
	}

	var config Config
	if err := common.UnmarshalJsonStr(raw, &config); err != nil {
		return DefaultConfig(), err
	}
	return Normalize(config), nil
}

func Validate(raw string) error {
	_, err := Parse(raw)
	return err
}

func Serialize(config Config) (string, error) {
	config = Normalize(config)
	if config.Models == nil {
		config.Models = []string{}
	}
	data, err := common.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetConfig() Config {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[OptionKey]
	common.OptionMapRWMutex.RUnlock()

	config, err := Parse(raw)
	if err != nil {
		return DefaultConfig()
	}
	return config
}

