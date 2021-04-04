package silencer

import (
	"bytes"
	"time"

	"gopkg.in/yaml.v2"
)

type RenderableMaintenance struct {
	Maintenance YamlMaintenance `yaml:"maintenance"`
	Next        time.Time       `yaml:"next"`
	IsActive    bool            `yaml:"isActive"`
}

type watchedMaintenanceStorage interface {
	WatchedMaintenances() []WatchedMaintenance
}

type StatusBoard struct {
	watchedMaintenanceStorage watchedMaintenanceStorage
	yamlMaintenanceIndex      YamlMaintenanceIndex
}

func NewStatusBoard(
	watchedMaintenanceStorage watchedMaintenanceStorage,
	yamlMaintenanceIndex YamlMaintenanceIndex,
) *StatusBoard {
	return &StatusBoard{
		watchedMaintenanceStorage,
		yamlMaintenanceIndex,
	}
}

func (b *StatusBoard) Render() ([]byte, error) {
	buf := bytes.Buffer{}
	yamlEncoder := yaml.NewEncoder(&buf)

	maintenances := b.watchedMaintenanceStorage.WatchedMaintenances()
	for _, m := range maintenances {
		err := yamlEncoder.Encode(RenderableMaintenance{
			Maintenance: b.yamlMaintenanceIndex[m.Maintenance.Hash],
			Next:        m.Next,
			IsActive:    m.IsActive,
		})
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
