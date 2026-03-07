package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/ReilBleem13/internal/domain"
	"github.com/ReilBleem13/internal/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestContainers struct {
	PostgresContainer *postgres.PostgresContainer
	PostgresDB        *sqlx.DB
}

var testContainers *TestContainers

func TestMain(m *testing.M) {
	ctx := context.Background()

	tc, err := setupContainers(ctx)
	if err != nil {
		log.Fatalf("failed to setup containers: %v", err)
	}

	testContainers = tc

	code := m.Run()

	tc.Cleanup(ctx)

	os.Exit(code)
}

func setupContainers(ctx context.Context) (*TestContainers, error) {
	tc := &TestContainers{}

	pgContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, err
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	db, err := sqlx.ConnectContext(ctx, "postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := runMigrations(db.DB); err != nil {
		return nil, err
	}

	tc.PostgresContainer = pgContainer
	tc.PostgresDB = db

	return tc, nil
}

func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "../infra/database/migrations/")
}

func (tc *TestContainers) Cleanup(ctx context.Context) error {
	if tc.PostgresDB != nil {
		if err := tc.PostgresDB.DB.Close(); err != nil {
			return err
		}
	}

	if tc.PostgresContainer != nil {
		if err := tc.PostgresContainer.Terminate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (tc *TestContainers) Reset(t *testing.T, ctx context.Context) {
	_, err := tc.PostgresDB.ExecContext(ctx, "TRUNCATE TABLE subs RESTART IDENTITY CASCADE")
	require.NoError(t, err, "failed to truncate subs table")
}

func DB() *sqlx.DB {
	return testContainers.PostgresDB
}

func newDate(t time.Time) domain.Date {
	y, m, d := t.Date()
	return domain.Date(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

func TestSubService_Create(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	req := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
		EndData:     nil,
	}

	createdSub, err := service.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, createdSub)
	require.Greater(t, createdSub.ID, int64(0))

	require.Equal(t, "YANDEX", createdSub.ServiceName)
	require.Equal(t, 400, createdSub.Price)
	require.Equal(t, userID, createdSub.UserID)
	require.Equal(t, startDate, createdSub.StartDate)
	require.Nil(t, createdSub.EndDate)

	var count int
	require.NoError(t, DB().GetContext(ctx, &count, `SELECT COUNT(*) FROM subs`))
	require.Equal(t, 1, count)

	var savedSub domain.Sub
	require.NoError(t, DB().GetContext(ctx, &savedSub, `SELECT * FROM subs WHERE id = $1`, createdSub.ID))
	require.Equal(t, createdSub.ID, savedSub.ID)
	require.Equal(t, "YANDEX", savedSub.ServiceName)
	require.Equal(t, 400, savedSub.Price)
	require.Equal(t, userID, savedSub.UserID)
	require.Equal(t, startDate, savedSub.StartDate)
}

func TestSubService_Create_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	req := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
		EndData:     nil,
	}

	_, err := service.Create(ctx, req)
	require.NoError(t, err)

	req2 := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
		EndData:     nil,
	}
	_, err = service.Create(ctx, req2)

	repoErr, ok := errors.AsType[*domain.AppError](err)
	require.True(t, ok)
	require.Equal(t, repoErr.Code, domain.ErrAlreadyExist().Code)
}

func TestSubService_Get(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	req := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
	}

	_, err := DB().ExecContext(t.Context(), `
		INSERT INTO subs (
			service_name,
			price,
			user_id,
			start_date
		) VALUES ($1, $2, $3, $4)	 
	`, req.ServiceName, req.Price, req.UserID, req.StartDate)
	require.NoError(t, err)

	gotSub, err := service.Get(t.Context(), "1")
	require.NoError(t, err)

	require.Equal(t, gotSub.ID, int64(1))
	require.Equal(t, gotSub.ServiceName, req.ServiceName)
	require.Equal(t, gotSub.UserID, req.UserID)
	require.Equal(t, gotSub.Price, req.Price)
	require.Equal(t, gotSub.StartDate, req.StartDate)
}

func TestSubService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	_, err := service.Get(t.Context(), "1")
	repoErr, ok := errors.AsType[*domain.AppError](err)
	require.True(t, ok)
	require.Equal(t, repoErr.Code, domain.ErrNotFound().Code)
}

func TestSubService_Update(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	req := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
	}

	_, err := DB().ExecContext(t.Context(), `
		INSERT INTO subs (
			service_name,
			price,
			user_id,
			start_date
		) VALUES ($1, $2, $3, $4)	 
	`, req.ServiceName, req.Price, req.UserID, req.StartDate)
	require.NoError(t, err)

	newEndDate := newDate(time.Now())
	updatedSub, err := service.Update(ctx, "1", &UpdateSubRequest{
		ServiceName: new("KION"),
		EndDate:     &newEndDate,
	})
	require.NoError(t, err)

	require.Equal(t, updatedSub.ID, int64(1))
	require.Equal(t, updatedSub.ServiceName, "KION")
	require.Equal(t, updatedSub.UserID, req.UserID)
	require.Equal(t, updatedSub.Price, req.Price)
	require.Equal(t, updatedSub.StartDate, req.StartDate)
	require.Equal(t, updatedSub.EndDate, &newEndDate)
}

func TestSubService_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	_, err := service.Update(ctx, "1", &UpdateSubRequest{
		ServiceName: new("KION"),
	})
	require.Error(t, err)
}

func TestSubService_Delete(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	req := &CreateSubRequest{
		ServiceName: "YANDEX",
		Price:       400,
		UserID:      userID,
		StartDate:   startDate,
	}

	_, err := DB().ExecContext(t.Context(), `
		INSERT INTO subs (
			service_name,
			price,
			user_id,
			start_date
		) VALUES ($1, $2, $3, $4)	 
	`, req.ServiceName, req.Price, req.UserID, req.StartDate)
	require.NoError(t, err)

	require.NoError(t, service.Delete(ctx, "1"))

	var count int
	require.NoError(t, DB().GetContext(ctx, &count, `SELECT COUNT(*) FROM subs`))
	require.Equal(t, count, 0)
}

func TestSubService_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	err := service.Delete(ctx, "1")
	repoErr, ok := errors.AsType[*domain.AppError](err)
	require.True(t, ok)
	require.Equal(t, repoErr.Code, domain.ErrNotFound().Code)
}

func TestSubService_List(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	req1 := &CreateSubRequest{
		ServiceName: "Kinosearch",
		Price:       400,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Now()),
	}
	createSub(t, req1)

	req2 := &CreateSubRequest{
		ServiceName: "Yandex",
		Price:       500,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Now()),
	}
	createSub(t, req2)

	resp1, err := service.List(ctx, "1", "1")
	require.NoError(t, err)
	require.Equal(t, 1, len(resp1.Content))
	require.Equal(t, 1, resp1.PageNumber)
	require.Equal(t, 1, resp1.PageSize)
	require.Equal(t, 2, resp1.TotalCount)

	resp2, err := service.List(ctx, "2", "1")
	require.NoError(t, err)
	require.Equal(t, 2, len(resp2.Content))
	require.Equal(t, 1, resp2.PageNumber)
	require.Equal(t, 2, resp2.PageSize)
	require.Equal(t, 2, resp2.TotalCount)

	resp3, err := service.List(ctx, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, len(resp3.Content))
	require.Equal(t, 1, resp3.PageNumber)
	require.Equal(t, 1, resp3.PageSize)
	require.Equal(t, 2, resp3.TotalCount)
}

func TestSubService_TotalAmount(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	req1 := &CreateSubRequest{
		ServiceName: "Kinosearch",
		Price:       500,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Now()),
	}
	createSub(t, req1)

	req2 := &CreateSubRequest{
		ServiceName: "Yandex",
		Price:       750,
		UserID:      req1.UserID,
		StartDate:   newDate(time.Now().Add(time.Hour * 24 * 7)),
	}
	createSub(t, req2)

	req3 := &CreateSubRequest{
		ServiceName: "Yandex",
		Price:       320,
		UserID:      req1.UserID,
		StartDate:   newDate(time.Now()),
	}
	createSub(t, req3)

	req4 := &CreateSubRequest{
		ServiceName: "Youtube",
		Price:       315,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Now()),
	}
	createSub(t, req4)

	resp1, err := service.TotalAmount(ctx, &TotalAmountRequest{
		RawUserID: req1.UserID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, req1.Price+req2.Price+req3.Price, resp1.Sum)

	resp2, err := service.TotalAmount(ctx, &TotalAmountRequest{
		RawUserID:  req1.UserID.String(),
		RawSubname: "Yandex",
	})
	require.NoError(t, err)
	require.Equal(t, req2.Price+req3.Price, resp2.Sum)

	resp3, err := service.TotalAmount(ctx, &TotalAmountRequest{
		RawUserID:  req1.UserID.String(),
		RawSubname: "Yandex",
		RawFrom:    "2026-03-09",
	})
	require.NoError(t, err)
	require.Equal(t, req2.Price, resp3.Sum)

	resp4, err := service.TotalAmount(ctx, &TotalAmountRequest{
		RawSubname: "Yandex",
	})
	require.NoError(t, err)
	require.Equal(t, req2.Price+req3.Price, resp4.Sum)

	resp5, err := service.TotalAmount(ctx, &TotalAmountRequest{})
	require.NoError(t, err)
	require.Equal(t, req1.Price+req2.Price+req3.Price+req4.Price, resp5.Sum)
}

func createSub(t *testing.T, req *CreateSubRequest) {
	t.Helper()

	_, err := DB().ExecContext(t.Context(), `
		INSERT INTO subs (
			service_name,
			price,
			user_id,
			start_date,
			end_date
		) VALUES ($1, $2, $3, $4, $5)
	`, req.ServiceName, req.Price, req.UserID, req.StartDate, req.EndData)
	require.NoError(t, err)
}
