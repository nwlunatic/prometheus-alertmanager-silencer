package silencer

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"
	"github.com/prometheus/alertmanager/pkg/labels"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/nwlunatic/prometheus-alertmanager-silencer/src/silencer"
	"github.com/nwlunatic/prometheus-alertmanager-silencer/tests/infrastructure"
)

func TestMaintenanceService(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(40 * time.Second)
	clockMock := silencer.ClockMock{
		now,
	}

	maintenance1 := silencer.YamlMaintenance{
		[]string{"alertname=test1"},
		"* * * * *",
		"50s",
	}

	maintenance2 := silencer.YamlMaintenance{
		[]string{"alertname=test2"},
		"* * * * *",
		"20s",
	}

	m1 := silencer.MustMaintenance(silencer.ParseMaintenance(maintenance1))
	m2 := silencer.MustMaintenance(silencer.ParseMaintenance(maintenance2))

	silence1 := silencer.Silence{
		m1.Matchers,
		m1.Duration,
		m1.Hash.String(),
		"maintenance service",
	}

	silence2 := silencer.Silence{
		mustTypeMatchers(cli.TypeMatchers([]labels.Matcher{
			mustParseMatcher(labels.ParseMatcher("alertname=test1")),
		})),
		time.Minute,
		uuid.NewV4().String(),
		"maintenance service",
	}

	silence3 := silencer.Silence{
		mustTypeMatchers(cli.TypeMatchers([]labels.Matcher{
			mustParseMatcher(labels.ParseMatcher("alertname=test1")),
		})),
		time.Minute,
		"other comment",
		"other author",
	}

	testCases := []struct {
		name                        string
		injectSilences              []silencer.Silence
		maintenances                []silencer.YamlMaintenance
		expectedSilenceComments     []string
		expectedWatchedMaintenances []silencer.WatchedMaintenance
	}{
		{
			name: "two active silences in alertmanager and two maintenances, one of which associated with silence. " +
				"should recover state (1 silence <-> 1 active maintenance) and delete extra silence",
			injectSilences: []silencer.Silence{
				silence1, silence2, silence3,
			},
			maintenances: []silencer.YamlMaintenance{
				maintenance1, maintenance2,
			},
			expectedSilenceComments: []string{silence1.Comment, silence3.Comment},
			expectedWatchedMaintenances: []silencer.WatchedMaintenance{
				{
					m1,
					m1.Schedule.Next(time.Now()),
					true,
				},
				{
					m2,
					m2.Schedule.Next(time.Now()),
					false,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amClient := infrastructure.SetupAlertManagerClient(t)
			infrastructure.DeleteActiveSilences(t, amClient)

			silenceService := silencer.NewSilenceService(
				amClient.Silence,
				clockMock,
			)
			ctx := context.Background()

			for _, s := range tc.injectSilences {
				_, err := silenceService.Add(ctx, s)
				if err != nil {
					t.Fatal(err)
				}
			}

			logger := logrus.New()
			maintenanceService := silencer.NewMaintenanceService(
				"maintenance service",
				silencer.MustMaintenances(silencer.ParseMaintenances(tc.maintenances)),
				silencer.NewActiveMaintenanceStorage(),
				silenceService,
				clockMock,
				logger,
			)
			err := maintenanceService.Start()
			if err != nil {
				t.Fatal(err)
			}
			defer maintenanceService.Stop()

			activeSilences := infrastructure.GetActiveSilences(t, amClient)
			comments := make([]string, 0)
			for _, s := range activeSilences {
				comments = append(comments, *s.Comment)
			}

			// alertmanager api does not guarantee order of silences
			sort.Strings(comments)
			assert.Equal(t, tc.expectedSilenceComments, comments)
			assert.Equal(t, tc.expectedWatchedMaintenances, maintenanceService.WatchedMaintenances())
		})
	}
}

func mustTypeMatchers(matchers models.Matchers, err error) models.Matchers {
	if err != nil {
		panic(err)
	}

	return matchers
}

func mustParseMatcher(matcher *labels.Matcher, err error) labels.Matcher {
	if err != nil {
		panic(err)
	}

	return *matcher
}
