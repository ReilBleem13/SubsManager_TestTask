package service

import (
	"github.com/ReilBleem13/internal/domain"
	"github.com/ReilBleem13/internal/repository"
	"github.com/google/uuid"
)

func mapCreateSubToDomain(req *CreateSubRequest) *repository.SubCreate {
	return &repository.SubCreate{
		ServiceName: req.ServiceName,
		Price:       req.Price,
		UserID:      req.UserID,
		StartDate:   req.StartDate,
		EndDate:     *req.EndDate,
	}
}

func mapUpdateSubRequestToRepo(subID int64, in *UpdateSubRequest) *repository.SubUpdate {
	return &repository.SubUpdate{
		ID:          subID,
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   in.StartDate,
		EndDate:     in.EndDate,
	}
}

func mapFilterToRepo(userID *uuid.UUID, subName string, from, to *domain.Date) *repository.SubFilter {
	return &repository.SubFilter{
		ServiceName: subName,
		UserID:      userID,
		From:        from,
		To:          to,
	}
}
