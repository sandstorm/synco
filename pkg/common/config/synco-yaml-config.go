package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

const SyncoYamlFile = ".synco.yml"

// SyncoConfig is the main structure which is serialized from/to .synco.yml files. It contains
// client-side config (for `synco receive`).
type SyncoConfig struct {
	Hosts []SyncoHostConfig `yaml:"hosts"`
}

type SyncoHostConfig struct {
	BaseUrl string `yaml:"baseUrl"`
}

func ReadFromYaml() (SyncoConfig, error) {
	var syncoConfig SyncoConfig
	file, err := os.ReadFile(SyncoYamlFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// we return the empty synco config
			return syncoConfig, nil
		}
		return syncoConfig, err
	}
	err = yaml.Unmarshal(file, &syncoConfig)
	if err != nil {
		return syncoConfig, fmt.Errorf("malformed YAML in %s - delete the file and try again: %w", SyncoYamlFile, err)
	}
	return syncoConfig, nil
}

func WriteToFile(syncoConfig SyncoConfig) error {
	bytes, err := yaml.Marshal(syncoConfig)
	if err != nil {
		return fmt.Errorf("config cannot be serialized: %w", err)
	}

	err = os.WriteFile(SyncoYamlFile, bytes, 0755)
	if err != nil {
		return fmt.Errorf("cannot write config file %s: %w", SyncoYamlFile, err)
	}

	return nil
}
