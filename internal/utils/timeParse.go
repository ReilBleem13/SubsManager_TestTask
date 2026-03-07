package utils

import (
	"time"

	"github.com/ReilBleem13/internal/domain"
)

func ParseDate(raw string) (*domain.Date, error) {
	if raw == "" {
		return nil, nil
	}

	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}

	d := domain.Date(t)
	return &d, nil
}
