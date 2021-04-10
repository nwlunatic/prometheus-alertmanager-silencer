package silencer

import (
	"context"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
)

type Silence struct {
	Matchers  models.Matchers
	StartAt   time.Time
	Duration  time.Duration
	Comment   string
	CreatedBy string
}

type ActiveSilenceID string

func (id ActiveSilenceID) strfmtUUID() strfmt.UUID {
	return strfmt.UUID(id)
}

type SilenceService struct {
	silenceClient *silence.Client
}

func NewSilenceService(
	silenceClient *silence.Client,
) *SilenceService {
	return &SilenceService{
		silenceClient,
	}
}

func (s *SilenceService) Add(ctx context.Context, silence Silence) (ActiveSilenceID, error) {
	startsAt := silence.StartAt.UTC()
	endsAt := startsAt.Add(silence.Duration)

	start := strfmt.DateTime(startsAt)
	end := strfmt.DateTime(endsAt)

	postableSilence := &models.PostableSilence{
		Silence: models.Silence{
			Matchers:  silence.Matchers,
			StartsAt:  &start,
			EndsAt:    &end,
			CreatedBy: &silence.CreatedBy,
			Comment:   &silence.Comment,
		},
	}

	return s.add(ctx, postableSilence)
}

func (s *SilenceService) Delete(ctx context.Context, id ActiveSilenceID) error {
	_, err := s.silenceClient.DeleteSilence(
		silence.NewDeleteSilenceParams().
			WithContext(ctx).
			WithSilenceID(id.strfmtUUID()),
	)
	if err != nil {
		return err
	}

	return nil
}

type ActiveSilence struct {
	ID      ActiveSilenceID
	Comment string
}

func (s *SilenceService) ActiveSilences(ctx context.Context, createdBy string) ([]ActiveSilence, error) {
	params := silence.NewGetSilencesParams().
		WithContext(ctx)
	silencesResp, err := s.silenceClient.GetSilences(params)
	if err != nil {
		return nil, err
	}

	activeSilences := make([]ActiveSilence, 0)
	for _, gettableSilence := range silencesResp.GetPayload() {
		if IsExpired(gettableSilence) {
			continue
		}

		if gettableSilence.Comment == nil {
			continue
		}

		if gettableSilence.ID == nil {
			continue
		}

		if !CreatedBy(gettableSilence, createdBy) {
			continue
		}

		activeSilences = append(activeSilences, ActiveSilence{
			ActiveSilenceID(*gettableSilence.ID),
			*gettableSilence.Comment,
		})
	}

	return activeSilences, nil
}

func (s *SilenceService) add(ctx context.Context, postableSilence *models.PostableSilence) (ActiveSilenceID, error) {
	silenceParams := silence.NewPostSilencesParams().
		WithContext(ctx).
		WithSilence(postableSilence)

	postOk, err := s.silenceClient.PostSilences(silenceParams)
	if err != nil {
		return "", err
	}

	id := ActiveSilenceID(postOk.Payload.SilenceID)

	return id, nil
}

func IsExpired(silence *models.GettableSilence) bool {
	return silence.Status != nil && *silence.Status.State == models.SilenceStatusStateExpired
}

func CreatedBy(silence *models.GettableSilence, createdBy string) bool {
	return silence.CreatedBy != nil && *silence.CreatedBy == createdBy
}
