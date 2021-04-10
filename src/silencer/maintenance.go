package silencer

import (
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

type MaintenanceHash uuid.UUID

func (hash MaintenanceHash) String() string {
	return uuid.UUID(hash).String()
}

type Maintenance struct {
	Hash     MaintenanceHash
	Matchers models.Matchers
	Schedule cron.Schedule
	Duration time.Duration
}

func (m Maintenance) ActiveAt(t time.Time) (bool, time.Time) {
	durationTimeAgo := t.Add(-m.Duration)
	startAt := m.Schedule.Next(durationTimeAgo)
	return startAt.Before(t), startAt
}
