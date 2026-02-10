package usecase

import (
	"context"
	"errors"

	"github.com/NasaVasa/botty/internal/domain"
)

type EventUsecase struct {
	gamma domain.GammaClient
}

func NewEventUsecase(gamma domain.GammaClient) *EventUsecase {
	return &EventUsecase{gamma: gamma}
}

func (u *EventUsecase) GetEvent(ctx context.Context, eventSlug string) (*domain.EventMarkets, error) {
	event, err := u.gamma.GetEventBySlug(ctx, eventSlug)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return event, nil
}
