package main

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/alertmanager/cli"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/nwlunatic/prometheus-alertmanager-silencer/src/silencer"
)

func main() {
	logger := logrus.New()

	cfg := parseFlags()

	configFile, err := os.Open(cfg.configFile)
	if err != nil {
		logger.Fatal(err)
	}

	yamlConfig, err := silencer.ParseYaml(configFile)
	if err != nil {
		logger.Fatal(err)
	}

	yamlMaintenanceIndex := silencer.BuildYamlMaintenanceIndex(yamlConfig.Maintenances)

	config, err := silencer.ConfigFromYaml(yamlConfig)
	if err != nil {
		logger.Fatal(err)
	}

	u, err := url.ParseRequestURI(cfg.alertManagerURL)
	if err != nil {
		logger.Fatal(err)
	}

	clock := silencer.Clock{}
	maintenanceService := silencer.NewMaintenanceService(
		"maintenance service",
		config.Maintenances,
		silencer.NewActiveMaintenanceStorage(),
		silencer.NewSilenceService(
			cli.NewAlertmanagerClient(u).Silence,
		),
		clock,
		logger,
	)
	err = maintenanceService.Start()
	if err != nil {
		logger.Fatal(err)
	}
	defer maintenanceService.Stop()

	statusBoardHandler := silencer.NewStatusBoardHandler(
		silencer.NewStatusBoard(
			maintenanceService,
			yamlMaintenanceIndex,
		),
	)

	errChan := make(chan error, 10)
	r := chi.NewRouter()
	r.Get("/", statusBoardHandler.Handle())
	go func() {
		errChan <- http.ListenAndServe(net.JoinHostPort("", "5000"), r)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sig:
			return
		case err := <-errChan:
			if err != nil {
				logger.Fatal(err)
			}
		}
	}
}

// cliFlags is a union of the fields, which application could parse from CLI args
type cliFlags struct {
	configFile      string
	alertManagerURL string
}

// parseFlags maps CLI flags to struct
func parseFlags() *cliFlags {
	cfg := cliFlags{}

	kingpin.Flag("config.file", "Config file").
		Envar("CONFIG_FILE").
		Default("silencer.yml").
		StringVar(&cfg.configFile)

	kingpin.Flag("alertmanager.url", "AlertManager url").
		Envar("ALERT_MANAGER_URL").
		Default("http://localhost:9093").
		StringVar(&cfg.alertManagerURL)

	kingpin.Parse()
	return &cfg
}
