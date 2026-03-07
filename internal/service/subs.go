package service

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/ReilBleem13/internal/domain"
	"github.com/ReilBleem13/internal/repository"
	"github.com/ReilBleem13/internal/utils"
	"github.com/google/uuid"
)

type SubService interface {
	Create(ctx context.Context, req *CreateSubRequest) (*domain.Sub, error)
	Get(ctx context.Context, rawSubID string) (*domain.Sub, error)
	Update(ctx context.Context, rawSubID string, req *UpdateSubRequest) (*domain.Sub, error)
	Delete(ctx context.Context, rawSubID string) error
	List(ctx context.Context, rawLimit, rawPage string) (*ListResponse, error)
	TotalAmount(ctx context.Context, req *TotalAmountRequest) (*TotalAmountResponse, error)
}

type sub struct {
	repo   repository.SubcRepo
	logger *slog.Logger
}

func NewSub(repo repository.SubcRepo, logger *slog.Logger) SubService {
	return &sub{
		repo:   repo,
		logger: logger,
	}
}

func (s *sub) Create(ctx context.Context, req *CreateSubRequest) (*domain.Sub, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.Create")

	req.ServiceName = strings.ToLower(req.ServiceName)

	createdSub := mapCreateSubToDomain(req)

	if err := s.repo.Create(ctx, createdSub); err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Create",
				"request", req,
				"error", err,
			)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Sub successfully created", "sub_id", createdSub.ID)
	return createdSub, nil
}

func (s *sub) Get(ctx context.Context, rawSubID string) (*domain.Sub, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.Get", "sub_id", rawSubID)

	subID, err := strconv.Atoi(rawSubID)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid subID")
		return nil, domain.ErrBadRequest()
	}

	sub, err := s.repo.Get(ctx, int64(subID))
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Get", "error", err)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Sub successfully got")
	return sub, nil
}

func (s *sub) Update(ctx context.Context, rawSubID string, req *UpdateSubRequest) (*domain.Sub, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.Update", "sub_id", rawSubID)

	subID, err := strconv.Atoi(rawSubID)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid subID")
		return nil, domain.ErrBadRequest()
	}

	updatedSub, err := s.repo.Update(ctx, mapUpdateSubRequestToRepo(int64(subID), req))
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Update",
				"request", req,
				"error", err,
			)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Sub successfully updated")
	return updatedSub, nil
}

func (s *sub) Delete(ctx context.Context, rawSubID string) error {
	s.logger.DebugContext(ctx, "Start srv.sub.Delete", "sub_id", rawSubID)

	subID, err := strconv.Atoi(rawSubID)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid subID")
		return domain.ErrBadRequest()
	}

	if err := s.repo.Delete(ctx, int64(subID)); err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Delete", "error", err)
		}
		return err
	}

	s.logger.InfoContext(ctx, "Sub successfully deleted")
	return nil
}

func (s *sub) List(ctx context.Context, rawLimit, rawPage string) (*ListResponse, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.List")

	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit < 1 {
		limit = 1
	}

	page, err := strconv.Atoi(rawPage)
	if err != nil || page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	subs, totalSubs, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.List", "error", err)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Subs successfully listed", "len", len(subs))
	return &ListResponse{
		PageNumber: page,
		PageSize:   len(subs),
		TotalCount: totalSubs,
		Content:    subs,
	}, nil
}

func (s *sub) TotalAmount(ctx context.Context, req *TotalAmountRequest) (*TotalAmountResponse, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.TotalAmount",
		"req", req,
	)

	var userID *uuid.UUID
	if req.RawUserID != "" {
		id, err := uuid.Parse(req.RawUserID)
		if err != nil {
			return nil, domain.ErrBadRequest().WithMessage("Invalid userID")
		}
		userID = &id
	}

	from, err := utils.ParseDate(req.RawFrom)
	if err != nil {
		return nil, domain.ErrBadRequest().WithMessage("Invalid fromDate")
	}

	to, err := utils.ParseDate(req.RawTo)
	if err != nil {
		return nil, domain.ErrBadRequest().WithMessage("Invalid endDate")
	}

	subName := strings.ToLower(req.RawSubname)

	totalSum, err := s.repo.TotalAmount(ctx, mapFilterToRepo(userID, subName, from, to))
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.TotalAmount", "error", err)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Total amount successfully got", "total_amount", totalSum)
	return &TotalAmountResponse{Sum: totalSum}, nil
}
