package silencer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatusBoard_Render(t *testing.T) {
	maintenance1 := YamlMaintenance{
		[]string{"alertname=test1"},
		"* * * * *",
		"50s",
	}

	maintenance2 := YamlMaintenance{
		[]string{"alertname=test2"},
		"6 * * * *",
		"30m",
	}

	m1 := MustMaintenance(ParseMaintenance(maintenance1))
	m2 := MustMaintenance(ParseMaintenance(maintenance2))

	yamlMaintenanceIndex := BuildYamlMaintenanceIndex([]YamlMaintenance{maintenance1, maintenance2})

	now := time.Now()

	testCases := []struct {
		name                      string
		watchedMaintenanceStorage watchedMaintenanceStorage
		expectedStatusBoardRender []byte
	}{
		{
			name:                      "no maintenances",
			watchedMaintenanceStorage: watchedMaintenanceStorageMock{},
			expectedStatusBoardRender: nil,
		},
		{
			name: "one active maintenances",
			watchedMaintenanceStorage: watchedMaintenanceStorageMock{
				items: []WatchedMaintenance{
					{
						m1,
						m1.Schedule.Next(now),
						true,
					},
				},
			},
			expectedStatusBoardRender: []byte(fmt.Sprintf(`maintenance:
  matchers:
  - alertname=test1
  schedule: '* * * * *'
  duration: 50s
next: %s
isActive: true
`, m1.Schedule.Next(now).Format(time.RFC3339))),
		},
		{
			name: "one active, one disabled maintenances",
			watchedMaintenanceStorage: watchedMaintenanceStorageMock{
				items: []WatchedMaintenance{
					{
						m1,
						m1.Schedule.Next(now),
						true,
					},
					{
						m2,
						m2.Schedule.Next(now),
						false,
					},
				},
			},
			expectedStatusBoardRender: []byte(fmt.Sprintf(`maintenance:
  matchers:
  - alertname=test1
  schedule: '* * * * *'
  duration: 50s
next: %s
isActive: true
---
maintenance:
  matchers:
  - alertname=test2
  schedule: 6 * * * *
  duration: 30m
next: %s
isActive: false
`, m1.Schedule.Next(now).Format(time.RFC3339), m2.Schedule.Next(now).Format(time.RFC3339))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			statusBoard := NewStatusBoard(tc.watchedMaintenanceStorage, yamlMaintenanceIndex)
			result, err := statusBoard.Render()
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.expectedStatusBoardRender, result)
		})
	}
}

type watchedMaintenanceStorageMock struct {
	items []WatchedMaintenance
}

func (m watchedMaintenanceStorageMock) WatchedMaintenances() []WatchedMaintenance {
	return m.items
}
