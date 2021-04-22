package main

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/nwlunatic/prometheus-alertmanager-silencer/src/httpserver"
	"github.com/nwlunatic/prometheus-alertmanager-silencer/src/signals"
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

	statusBoardHandler := silencer.NewStatusBoardHandler(
		silencer.NewStatusBoard(
			maintenanceService,
			yamlMaintenanceIndex,
		),
	)

	r := chi.NewRouter()
	r.Get("/", statusBoardHandler.Handle())

	server := httpserver.NewServer(&http.Server{Addr: net.JoinHostPort("", "5000"), Handler: r})
	serverErr := make(chan error)
	go func() {
		defer close(serverErr)
		serverErr <- server.Start()
	}()

	gracefulStopErrors := signals.BindGracefulStop(context.Background(), server, maintenanceService)
	errChan := joinErrorChannels(serverErr, gracefulStopErrors)
	for err := range errChan {
		if err != nil {
			logger.Fatal(err)
		}
	}
}

func joinErrorChannels(in ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup

	for _, e := range in {
		wg.Add(1)
		go func(e <-chan error) {
			for err := range e {
				out <- err
			}
			wg.Done()
		}(e)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
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
