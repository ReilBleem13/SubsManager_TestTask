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
	s.logger.DebugContext(ctx, "Start srv.sub.Create", "request", req)

	if err := s.validateCreate(ctx, req); err != nil {
		return nil, err
	}

	mappedReq := mapCreateSubToDomain(req)

	createdSub, err := s.repo.Create(ctx, mappedReq)
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Create",
				"error", err,
			)
		}
		return nil, err
	}

	s.logger.InfoContext(ctx, "Sub successfully created", "sub_id", createdSub.ID)
	return createdSub, nil
}

func (s *sub) validateCreate(ctx context.Context, req *CreateSubRequest) error {
	if req.ServiceName == "" {
		err := domain.ErrBadRequest().WithMessage("empty service name")
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return err
	}
	req.ServiceName = strings.ToLower(req.ServiceName)

	if req.Price < 0 {
		err := domain.ErrBadRequest().WithMessage("price less than 0")
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return err
	}

	if req.EndDate == nil {
		d := req.StartDate.AddMonths(1)
		req.EndDate = &d
	}

	if req.EndDate.Before(req.StartDate) {
		err := domain.ErrBadRequest().WithMessage("end date before start date")
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return err
	}
	return nil
}

func (s *sub) Get(ctx context.Context, rawSubID string) (*domain.Sub, error) {
	s.logger.DebugContext(ctx, "Start srv.sub.Get", "sub_id", rawSubID)

	subID, err := strconv.Atoi(rawSubID)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid subID")
		return nil, domain.ErrBadRequest().WithMessage("invalid sub id")
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
		return nil, domain.ErrBadRequest().WithMessage("invalid sub id")
	}

	if err := s.validateUpdate(ctx, int64(subID), req); err != nil {
		return nil, err
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

func (s *sub) validateUpdate(ctx context.Context, subID int64, req *UpdateSubRequest) error {
	oldSub, err := s.repo.Get(ctx, subID)
	if err != nil {
		if repoErr, ok := errors.AsType[*domain.AppError](err); ok {
			s.logger.WarnContext(ctx, repoErr.Message)
		} else {
			s.logger.ErrorContext(ctx, "Failed srv.sub.Get",
				"error", err,
			)
		}
		return err
	}

	if req.ServiceName != nil {
		if *req.ServiceName == "" {
			err := domain.ErrBadRequest().WithMessage("empty service name")
			s.logger.WarnContext(ctx, "Invalid request", "error", err)
			return err
		}

		*req.ServiceName = strings.ToLower(*req.ServiceName)
	}

	if req.Price != nil && *req.Price < 0 {
		err := domain.ErrBadRequest().WithMessage("price less than 0")
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return err
	}

	if req.StartDate != nil && req.EndDate != nil {
		if req.EndDate.Before(*req.StartDate) {
			err := domain.ErrBadRequest().WithMessage("end date before start date")
			s.logger.WarnContext(ctx, "Invalid request", "error", err)
			return err
		}
	} else if req.StartDate != nil {
		endDate := oldSub.EndDate
		if endDate.Before(*req.StartDate) {
			err := domain.ErrBadRequest().WithMessage("new start date after existing end date")
			s.logger.WarnContext(ctx, "Invalid request", "error", err)
			return err
		}
	} else if req.EndDate != nil {
		startDate := oldSub.StartDate
		if (*req.EndDate).Before(startDate) {
			err := domain.ErrBadRequest().WithMessage("new end date before existing start date")
			s.logger.WarnContext(ctx, "Invalid request", "error", err)
			return err
		}
	}
	return nil
}

func (s *sub) Delete(ctx context.Context, rawSubID string) error {
	s.logger.DebugContext(ctx, "Start srv.sub.Delete", "sub_id", rawSubID)

	subID, err := strconv.Atoi(rawSubID)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid subID")
		return domain.ErrBadRequest().WithMessage("invalid sub id")
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

	filter, err := s.validateTotalAmount(ctx, req)
	if err != nil {
		return nil, err
	}

	totalSum, err := s.repo.TotalAmount(ctx, filter)
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

func (s *sub) validateTotalAmount(ctx context.Context, req *TotalAmountRequest) (*repository.SubFilter, error) {
	var userID *uuid.UUID
	if req.RawUserID != "" {
		id, err := uuid.Parse(req.RawUserID)
		if err != nil {
			s.logger.WarnContext(ctx, "Invalid request", "error", err)
			return nil, domain.ErrBadRequest().WithMessage("invalid userID")
		}
		userID = &id
	}

	from, err := utils.ParseDate(req.RawFrom)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return nil, domain.ErrBadRequest().WithMessage("invalid from date")
	}

	to, err := utils.ParseDate(req.RawTo)
	if err != nil {
		s.logger.WarnContext(ctx, "Invalid request", "error", err)
		return nil, domain.ErrBadRequest().WithMessage("invalid to Date")
	}

	var subName string
	if req.RawSubname != "" {
		subName = strings.ToLower(req.RawSubname)
	}
	return mapFilterToRepo(userID, subName, from, to), nil
}
