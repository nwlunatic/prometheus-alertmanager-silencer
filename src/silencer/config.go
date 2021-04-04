package silencer

import (
	"io"
	"time"

	"github.com/prometheus/alertmanager/cli"
	"github.com/prometheus/alertmanager/pkg/labels"
	"github.com/prometheus/common/model"
	"github.com/robfig/cron/v3"
)

type Config struct {
	Maintenances []Maintenance
}

func Parse(reader io.Reader) (Config, error) {
	yamlConfig, err := ParseYaml(reader)
	if err != nil {
		return Config{}, err
	}

	return ConfigFromYaml(yamlConfig)
}

func ConfigFromYaml(config YamlConfig) (Config, error) {
	maintenances, err := ParseMaintenances(config.Maintenances)
	if err != nil {
		return Config{}, err
	}

	c := Config{
		maintenances,
	}

	return c, nil
}

func ParseMaintenances(maintenances []YamlMaintenance) ([]Maintenance, error) {
	result := make([]Maintenance, len(maintenances))
	for i, m := range maintenances {
		var err error
		result[i], err = ParseMaintenance(m)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func ParseMaintenance(maintenance YamlMaintenance) (Maintenance, error) {
	matchers, err := parseMatchers(maintenance.Matchers)
	if err != nil {
		return Maintenance{}, err
	}

	typeMatchers, err := cli.TypeMatchers(matchers)
	if err != nil {
		return Maintenance{}, err
	}

	schedule, err := cron.ParseStandard(maintenance.Schedule)
	if err != nil {
		return Maintenance{}, err
	}

	d, err := model.ParseDuration(maintenance.Duration)
	if err != nil {
		return Maintenance{}, err
	}
	duration := time.Duration(d)

	return Maintenance{
		maintenance.Hash(),
		typeMatchers,
		schedule,
		duration,
	}, nil
}

func parseMatchers(inputMatchers []string) ([]labels.Matcher, error) {
	matchers := make([]labels.Matcher, 0, len(inputMatchers))

	for _, v := range inputMatchers {
		matcher, err := labels.ParseMatcher(v)
		if err != nil {
			return []labels.Matcher{}, err
		}

		matchers = append(matchers, *matcher)
	}

	return matchers, nil
}

func MustMaintenances(maintenances []Maintenance, err error) []Maintenance {
	if err != nil {
		panic(err)
	}

	return maintenances
}

func MustMaintenance(maintenance Maintenance, err error) Maintenance {
	if err != nil {
		panic(err)
	}

	return maintenance
}
