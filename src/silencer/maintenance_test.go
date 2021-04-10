package silencer

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

func TestMaintenance_IsActiveAt(t *testing.T) {
	testCases := []struct {
		maintenance Maintenance
		at          time.Time
		isActive    bool
		startAt     time.Time
	}{
		{
			maintenance: Maintenance{
				Schedule: mustParseSchedule(cron.ParseStandard("* * * * *")),
				Duration: 30 * time.Second,
			},
			at:       time.Time{}.Add(20 * time.Second),
			isActive: true,
			startAt:  time.Time{},
		},
		{
			maintenance: Maintenance{
				Schedule: mustParseSchedule(cron.ParseStandard("* * * * *")),
				Duration: 30 * time.Second,
			},
			at:       time.Time{}.Add(40 * time.Second),
			isActive: false,
			startAt:  time.Time{}.Add(60 * time.Second),
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			isActive, startAt := tc.maintenance.ActiveAt(tc.at)
			assert.Equal(t, tc.isActive, isActive)
			assert.Equal(t, tc.startAt, startAt)
		})
	}
}

func mustParseSchedule(s cron.Schedule, err error) cron.Schedule {
	if err != nil {
		panic(err)
	}

	return s
}
