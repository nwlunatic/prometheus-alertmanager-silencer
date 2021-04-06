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

func (m Maintenance) IsActiveAt(t time.Time) bool {
	durationTimeAgo := t.Add(-m.Duration)
	m.Schedule.Next(durationTimeAgo)
	return m.Schedule.Next(durationTimeAgo).Before(t)
}
