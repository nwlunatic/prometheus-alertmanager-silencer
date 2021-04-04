package infrastructure

import (
	"net/url"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"

	"github.com/nwlunatic/prometheus-alertmanager-silencer/src/silencer"
)

func SetupAlertManagerClient(t *testing.T) *client.Alertmanager {
	u, err := url.ParseRequestURI(
		GetWithDefault("ALERT_MANAGER_URL", "http://localhost:9093"),
	)
	if err != nil {
		t.Fatal(err)
	}

	return cli.NewAlertmanagerClient(u)
}

func GetActiveSilences(t *testing.T, amClient *client.Alertmanager) []*models.GettableSilence {
	resp, err := amClient.Silence.GetSilences(silence.NewGetSilencesParams())
	if err != nil {
		t.Fatal(err)
	}

	result := make([]*models.GettableSilence, 0)
	for _, gettableSilence := range resp.Payload {
		if silencer.IsExpired(gettableSilence) {
			continue
		}
		result = append(result, gettableSilence)
	}

	return result
}

func DeleteActiveSilences(t *testing.T, amClient *client.Alertmanager) {
	gettableSilences := GetActiveSilences(t, amClient)

	ids := make([]string, 0)
	for _, gettableSilence := range gettableSilences {
		ids = append(ids, *gettableSilence.ID)
	}

	for _, id := range ids {
		_, err := amClient.Silence.DeleteSilence(silence.NewDeleteSilenceParams().WithSilenceID(strfmt.UUID(id)))
		if err != nil {
			t.Fatal(err)
		}
	}
}
