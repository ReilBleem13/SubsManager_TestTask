package service

import (
	"github.com/ReilBleem13/internal/domain"
	"github.com/google/uuid"
)

type CreateSubRequest struct {
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartDate   domain.Date
	EndDate     *domain.Date
}

type UpdateSubRequest struct {
	ServiceName *string
	Price       *int
	UserID      *uuid.UUID
	StartDate   *domain.Date
	EndDate     *domain.Date
}

type TotalAmountRequest struct {
	RawUserID  string
	RawSubname string
	RawFrom    string
	RawTo      string
}

// Response
type ListResponse struct {
	PageNumber int          `json:"page_number"`
	PageSize   int          `json:"page_size"`
	TotalCount int          `json:"total_count"`
	Content    []domain.Sub `json:"result"`
}

type TotalAmountResponse struct {
	Sum int `json:"sum"`
}
