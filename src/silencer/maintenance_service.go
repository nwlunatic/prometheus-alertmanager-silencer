package silencer

import (
	"context"
	"time"

	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
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

	cron        *cron.Cron
	cronEntries map[int]cron.EntryID

	logger logrus.FieldLogger
}

func NewMaintenanceService(
	name string,
	maintenances []Maintenance,
	activeMaintenanceStorage activeMaintenanceStorage,
	silencer silencer,
	logger logrus.FieldLogger,
) *MaintenanceService {
	return &MaintenanceService{
		name,
		maintenances,
		activeMaintenanceStorage,
		silencer,
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
			s.addMaintenance(ctx, maintenance)
		}))

		s.cronEntries[i] = entryID
	}

	s.cron.Start()

	return nil
}

func (s *MaintenanceService) Stop() {
	s.cron.Stop()
}

type WatchedMaintenance struct {
	Maintenance Maintenance
	Next        time.Time
	IsActive    bool
}

func (s *MaintenanceService) WatchedMaintenances() []WatchedMaintenance {
	result := make([]WatchedMaintenance, len(s.maintenances))

	now := time.Now()
	for i, m := range s.maintenances {
		result[i] = WatchedMaintenance{
			Maintenance: m,
			IsActive:    s.activeMaintenanceStorage.IsActive(m.Hash),
			Next:        s.cron.Entry(s.cronEntries[i]).Schedule.Next(now),
		}
	}

	return result
}

func (s *MaintenanceService) addMaintenance(ctx context.Context, maintenance Maintenance) {
	silenceID, err := s.silencer.Add(ctx, Silence{
		maintenance.Matchers,
		maintenance.Duration,
		maintenance.Hash.String(),
		s.name,
	})
	if err != nil {
		s.logger.WithError(err).Infof("failed to post silence: %s", err.Error())
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

	for _, m := range s.maintenances {
		_, ok := activeMaintenanceIndex[m.Hash]
		if ok {
			s.activeMaintenanceStorage.Add(m.Hash)
			delete(activeMaintenanceIndex, m.Hash)
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

	return nil
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
