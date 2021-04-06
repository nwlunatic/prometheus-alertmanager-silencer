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
	}{
		{
			maintenance: Maintenance{
				Schedule: mustParseSchedule(cron.ParseStandard("* * * * *")),
				Duration: 30 * time.Second,
			},
			at:       time.Time{}.Add(20 * time.Second),
			isActive: true,
		},
		{
			maintenance: Maintenance{
				Schedule: mustParseSchedule(cron.ParseStandard("* * * * *")),
				Duration: 30 * time.Second,
			},
			at:       time.Time{}.Add(40 * time.Second),
			isActive: false,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.isActive, tc.maintenance.IsActiveAt(tc.at))
		})
	}
}

func mustParseSchedule(s cron.Schedule, err error) cron.Schedule {
	if err != nil {
		panic(err)
	}

	return s
}
