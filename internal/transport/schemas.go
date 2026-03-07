package transport

import (
	"github.com/ReilBleem13/internal/domain"
	"github.com/google/uuid"
)

type CreateSubJSON struct {
	ServiceName string       `json:"service_name"`
	Price       int          `json:"price"`
	UserID      uuid.UUID    `json:"user_id"`
	StartDate   domain.Date  `json:"start_date"`
	EndData     *domain.Date `json:"end_date"`
}

type UpdateSubJSON struct {
	ServiceName *string      `json:"service_name"`
	Price       *int         `json:"price"`
	UserID      *uuid.UUID   `json:"user_id"`
	StartDate   *domain.Date `json:"start_date"`
	EndData     *domain.Date `json:"end_date"`
}
