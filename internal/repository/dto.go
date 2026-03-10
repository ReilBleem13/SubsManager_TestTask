package repository

import (
	"github.com/ReilBleem13/internal/domain"
	"github.com/google/uuid"
)

type SubCreate struct {
	ServiceName string      `db:"service_name"`
	Price       int         `db:"price"`
	UserID      uuid.UUID   `db:"user_id"`
	StartDate   domain.Date `db:"start_date"`
	EndDate     domain.Date `db:"end_date"`
}

type SubUpdate struct {
	ID          int64        `db:"id"`
	ServiceName *string      `db:"service_name"`
	Price       *int         `db:"price"`
	UserID      *uuid.UUID   `db:"user_id"`
	StartDate   *domain.Date `db:"start_date"`
	EndDate     *domain.Date `db:"end_date"`
}

type SubFilter struct {
	UserID      *uuid.UUID
	ServiceName string
	From        *domain.Date
	To          *domain.Date
}
