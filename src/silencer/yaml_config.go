package silencer

import (
	"io"
	"strings"

	uuid "github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
)

type YamlMaintenance struct {
	Matchers []string `yaml:"matchers"`
	Schedule string   `yaml:"schedule"`
	Duration string   `yaml:"duration"`
}

func (m YamlMaintenance) Hash() MaintenanceHash {
	value := strings.Join(m.Matchers, ",") +
		m.Schedule +
		m.Duration

	return MaintenanceHash(uuid.NewV5(uuid.UUID{}, value))
}

type YamlConfig struct {
	Maintenances []YamlMaintenance `yaml:"maintenances,omitempty"`
}

func ParseYaml(reader io.Reader) (YamlConfig, error) {
	config := YamlConfig{}
	yamlDecoder := yaml.NewDecoder(reader)
	err := yamlDecoder.Decode(&config)
	if err != nil {
		return YamlConfig{}, err
	}

	return config, nil
}

type YamlMaintenanceIndex map[MaintenanceHash]YamlMaintenance

func BuildYamlMaintenanceIndex(maintenances []YamlMaintenance) YamlMaintenanceIndex {
	index := make(YamlMaintenanceIndex)

	for _, m := range maintenances {
		index[m.Hash()] = m
	}

	return index
}
