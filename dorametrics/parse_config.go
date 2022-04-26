package dorametrics

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

func parseConfig(configPath string, config *ControllerConfig, lookup *map[string]bool) error {
	byteArray, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("can't read configuration file %s: %v", configPath, err)
	}

	jsonArray, err := yaml.YAMLToJSON(byteArray)
	if err != nil {
		return fmt.Errorf("can't convert configuration file %s to JSON: %v", configPath, err)
	}

	err = json.Unmarshal(jsonArray, config)
	if err != nil {
		return fmt.Errorf("can't unmarshal configuration file %s to internal data structure: %v", configPath, err)
	}

	// populate lookup
	for _, target := range config.Targets {
		key := target.Namespace + target.Name
		(*lookup)[key] = true
	}

	return nil
}
