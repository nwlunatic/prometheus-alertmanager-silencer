package silencer

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type activeMaintenanceStorage interface {
	Add(hash MaintenanceHash)
	Delete(hash MaintenanceHash)
	IsActive(hash MaintenanceHash) bool
}

type silencer interface {
	Add(ctx context.Context, silence Silence) (ActiveSilenceID, error)
	Delete(ctx context.Context, id ActiveSilenceID) error
	ActiveSilences(ctx context.Context, createdBy string) ([]ActiveSilence, error)
}

type MaintenanceService struct {
	name                     string
	maintenances             []Maintenance
	activeMaintenanceStorage activeMaintenanceStorage
	silencer                 silencer
	clock                    clock

	cron        *cron.Cron
	cronEntries map[int]cron.EntryID

	logger logrus.FieldLogger
}

func NewMaintenanceService(
	name string,
	maintenances []Maintenance,
	activeMaintenanceStorage activeMaintenanceStorage,
	silencer silencer,
	clock clock,
	logger logrus.FieldLogger,
) *MaintenanceService {
	return &MaintenanceService{
		name,
		maintenances,
		activeMaintenanceStorage,
		silencer,
		clock,
		cron.New(),
		make(map[int]cron.EntryID),
		logger,
	}
}

func (s *MaintenanceService) Start() error {
	ctx := context.Background()

	err := s.recoverState(ctx)
	if err != nil {
		return err
	}

	for i, maintenance := range s.maintenances {
		entryID := s.cron.Schedule(maintenance.Schedule, cron.FuncJob(func() {
			s.addMaintenance(ctx, maintenance, s.clock.Now())
		}))

		s.cronEntries[i] = entryID
	}

	s.cron.Start()

	return nil
}

func (s *MaintenanceService) Stop(ctx context.Context) error {
	stopCtx := s.cron.Stop()
	<-stopCtx.Done()
	err := stopCtx.Err()
	if errors.Cause(err) != context.Canceled {
		return err
	}

	return nil
}

type WatchedMaintenance struct {
	Maintenance Maintenance
	Next        time.Time
	IsActive    bool
}

func (s *MaintenanceService) WatchedMaintenances() []WatchedMaintenance {
	result := make([]WatchedMaintenance, len(s.maintenances))

	now := s.clock.Now()
	for i, m := range s.maintenances {
		result[i] = WatchedMaintenance{
			Maintenance: m,
			IsActive:    s.activeMaintenanceStorage.IsActive(m.Hash),
			Next:        s.cron.Entry(s.cronEntries[i]).Schedule.Next(now),
		}
	}

	return result
}

func (s *MaintenanceService) addMaintenance(ctx context.Context, maintenance Maintenance, startAt time.Time) {
	silenceID, err := s.silencer.Add(ctx, Silence{
		maintenance.Matchers,
		startAt,
		maintenance.Duration,
		maintenance.Hash.String(),
		s.name,
	})
	if err != nil {
		s.logger.WithError(err).Infof("failed to post silence: %s", err.Error())
		return
	}

	s.activeMaintenanceStorage.Add(maintenance.Hash)

	time.AfterFunc(maintenance.Duration, func() {
		err := s.silencer.Delete(ctx, silenceID)
		if err != nil {
			s.logger.WithError(err).Infof("failed to delete silence %s", silenceID)
		}

		s.activeMaintenanceStorage.Delete(maintenance.Hash)
	})
}

func (s *MaintenanceService) recoverState(ctx context.Context) error {
	activeSilences, err := s.silencer.ActiveSilences(ctx, s.name)
	if err != nil {
		return err
	}

	activeMaintenanceIndex, err := buildActiveMaintenanceIndex(activeSilences)
	if err != nil {
		return err
	}

	maintenancesWithoutSilences := make([]Maintenance, 0)
	for _, m := range s.maintenances {
		_, ok := activeMaintenanceIndex[m.Hash]
		if ok {
			s.activeMaintenanceStorage.Add(m.Hash)
			delete(activeMaintenanceIndex, m.Hash)
		} else {
			maintenancesWithoutSilences = append(maintenancesWithoutSilences, m)
		}
	}

	nonActualActiveSilenceIndex := buildActiveSilenceIndex(activeMaintenanceIndex)

	for _, silence := range activeSilences {
		_, ok := nonActualActiveSilenceIndex[silence.ID]
		if ok {
			err := s.silencer.Delete(ctx, silence.ID)
			if err != nil {
				return err
			}
		}
	}

	s.addMissingActiveMaintenances(ctx, maintenancesWithoutSilences)

	return nil
}

func (s *MaintenanceService) addMissingActiveMaintenances(ctx context.Context, maintenances []Maintenance) {
	now := s.clock.Now()
	for _, m := range maintenances {
		isActive, startAt := m.ActiveAt(now)
		if isActive {
			s.addMaintenance(ctx, m, startAt)
		}
	}
}

func buildActiveMaintenanceIndex(activeSilences []ActiveSilence) (map[MaintenanceHash]ActiveSilenceID, error) {
	result := make(map[MaintenanceHash]ActiveSilenceID)
	for _, s := range activeSilences {
		hash, err := uuid.FromString(s.Comment)
		if err != nil {
			return nil, err
		}

		result[MaintenanceHash(hash)] = s.ID
	}

	return result, nil
}

func buildActiveSilenceIndex(activeMaintenanceIndex map[MaintenanceHash]ActiveSilenceID) map[ActiveSilenceID]struct{} {
	result := make(map[ActiveSilenceID]struct{})
	for _, v := range activeMaintenanceIndex {
		result[v] = struct{}{}
	}

	return result
}
