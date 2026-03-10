package service

import (
	"context"
	"database/sql"
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

func TestSubServce_Create(t *testing.T) {
	ctx := context.Background()

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	endDate := startDate.AddMonths(1)

	tests := []struct {
		name         string
		prepare      func()
		req          *CreateSubRequest
		wantErr      bool
		wantErrCode  string
		validateFunc func(t *testing.T, req *CreateSubRequest, sub *domain.Sub)
	}{
		{
			name: "Success. Created",
			req: &CreateSubRequest{
				ServiceName: "YANDEX",
				Price:       400,
				UserID:      userID,
				StartDate:   newDate(time.Now()),
				EndDate:     nil,
			},
			validateFunc: func(t *testing.T, req *CreateSubRequest, sub *domain.Sub) {
				require.Equal(t, int64(1), sub.ID)
				require.Equal(t, req.UserID, sub.UserID)
				require.Equal(t, req.ServiceName, sub.ServiceName)
				require.Equal(t, req.Price, sub.Price)
				require.Equal(t, req.StartDate, sub.StartDate)
				require.Equal(t, req.StartDate.AddMonths(1), sub.EndDate)
				require.NotEmpty(t, sub.CreatedAt)
				require.NotEmpty(t, sub.UpdatedAt)

				require.Equal(t, 1, countSubs(t))
			},
		},
		{
			name: "Error. Already exist",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			req: &CreateSubRequest{
				ServiceName: "YANDEX",
				Price:       400,
				UserID:      userID,
				StartDate:   startDate,
				EndDate:     &endDate,
			},
			wantErr:     true,
			wantErrCode: "sub already exists",
		},
		{
			name: "Error. Empty service name",
			req: &CreateSubRequest{
				ServiceName: "",
				Price:       400,
				UserID:      userID,
				StartDate:   startDate,
				EndDate:     &endDate,
			},
			wantErr:     true,
			wantErrCode: "empty service name",
		},
		{
			name: "Error. Price less than 0",
			req: &CreateSubRequest{
				ServiceName: "yandex",
				Price:       -1,
				UserID:      userID,
				StartDate:   startDate,
				EndDate:     &endDate,
			},
			wantErr:     true,
			wantErrCode: "price less than 0",
		},
		{
			name: "Error. End date before start date",
			req: &CreateSubRequest{
				ServiceName: "yandex",
				Price:       400,
				UserID:      userID,
				StartDate:   endDate,
				EndDate:     &startDate,
			},
			wantErr:     true,
			wantErrCode: "end date before start date",
		},
	}

	for _, test := range tests {
		testContainers.Reset(t, ctx)
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			createdSub, err := service.Create(ctx, test.req)
			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, createdSub)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			if test.validateFunc != nil {
				test.validateFunc(t, test.req, createdSub)
			}
		})
	}
}

func TestSubService_Get(t *testing.T) {
	ctx := context.Background()

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	endDate := startDate.AddMonths(1)

	tests := []struct {
		name        string
		rawSubID    string
		prepare     func()
		wantErr     bool
		wantErrCode string
		validate    func(got *domain.Sub)
	}{
		{
			name:     "Success. Got",
			rawSubID: "1",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			validate: func(got *domain.Sub) {
				require.Equal(t, int64(1), got.ID)
				require.Equal(t, userID, got.UserID)
				require.Equal(t, 400, got.Price)
				require.Equal(t, "yandex", got.ServiceName)
				require.Equal(t, startDate, got.StartDate)
				require.Equal(t, endDate, got.EndDate)
				require.NotEmpty(t, got.CreatedAt)
				require.NotEmpty(t, got.UpdatedAt)
			},
		},
		{
			name:        "Error. Not found",
			rawSubID:    "1",
			wantErr:     true,
			wantErrCode: "sub not found",
		},
		{
			name:        "Error. Invalid SubID",
			rawSubID:    "-",
			wantErr:     true,
			wantErrCode: "invalid sub id",
		},
	}

	for _, test := range tests {
		testContainers.Reset(t, ctx)
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			got, err := service.Get(ctx, test.rawSubID)

			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			if test.validate != nil {
				test.validate(got)
			}
		})
	}
}

func TestSubService_Update(t *testing.T) {
	ctx := context.Background()

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()
	startDate := newDate(time.Now())
	endDate := startDate.AddMonths(1)

	tests := []struct {
		name            string
		prepare         func()
		reqRawSubID     string
		reqUpdateSubReq *UpdateSubRequest
		wantErr         bool
		wantErrCode     string
		validate        func(got *domain.Sub)
	}{
		{
			name: "Success. Updated",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				ServiceName: new("yandex-updated"),
			},
			validate: func(got *domain.Sub) {
				require.Equal(t, "yandex-updated", got.ServiceName)
				require.Equal(t, userID, got.UserID)
				require.Equal(t, 400, got.Price)
				require.Equal(t, startDate, got.StartDate)
				require.Equal(t, endDate, got.EndDate)
				require.NotEmpty(t, got.CreatedAt)
				require.NotEmpty(t, got.UpdatedAt)
			},
		},
		{
			name: "Success. Updated",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				ServiceName: new("yandex-updated"),
				Price:       new(800),
			},
			validate: func(got *domain.Sub) {
				require.Equal(t, "yandex-updated", got.ServiceName)
				require.Equal(t, userID, got.UserID)
				require.Equal(t, 800, got.Price)
				require.Equal(t, startDate, got.StartDate)
				require.Equal(t, endDate, got.EndDate)
				require.NotEmpty(t, got.CreatedAt)
				require.NotEmpty(t, got.UpdatedAt)
			},
		},
		{
			name:        "Error. Not found",
			reqRawSubID: "1",
			wantErr:     true,
			wantErrCode: "sub not found",
		},
		{
			name:        "Error. Invalid subID",
			reqRawSubID: "-",
			wantErr:     true,
			wantErrCode: "invalid sub id",
		},
		{
			name: "Error. Empty service name",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				ServiceName: new(string),
			},
			wantErr:     true,
			wantErrCode: "empty service name",
		},
		{
			name: "Error. Price less than 0",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				Price: new(-1),
			},
			wantErr:     true,
			wantErrCode: "price less than 0",
		},
		{
			name: "Error. End Date before start date",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				StartDate: &endDate,
				EndDate:   &startDate,
			},
			wantErr:     true,
			wantErrCode: "end date before start date",
		},
		{
			name: "Error. New start date after existing end date",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				StartDate: &[]domain.Date{newDate(time.Time(endDate).Add(24 * time.Hour))}[0],
			},
			wantErr:     true,
			wantErrCode: "new start date after existing end date",
		},
		{
			name: "Error. New end date before existing start date",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      userID,
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			reqRawSubID: "1",
			reqUpdateSubReq: &UpdateSubRequest{
				EndDate: &[]domain.Date{newDate(time.Date(2000, 0, 0, 0, 0, 0, 0, time.UTC))}[0],
			},
			wantErr:     true,
			wantErrCode: "new end date before existing start date",
		},
	}

	for _, test := range tests {
		testContainers.Reset(t, ctx)
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			got, err := service.Update(ctx, test.reqRawSubID, test.reqUpdateSubReq)

			if test.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			if test.validate != nil {
				test.validate(got)
			}
		})
	}
}

func TestSubService_Delete(t *testing.T) {
	ctx := context.Background()

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	tests := []struct {
		name        string
		prepare     func()
		reqRawSubID string
		wantErr     bool
		wantErrCode string
		validate    func()
	}{
		{
			name: "Success. Deleted",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   newDate(time.Now()),
					EndDate:     &[]domain.Date{newDate(time.Now())}[0],
				})
			},
			reqRawSubID: "1",
			validate: func() {
				require.Equal(t, 0, countSubs(t))
			},
		},
		{
			name:        "Error. Invalid subID",
			reqRawSubID: "-",
			wantErr:     true,
			wantErrCode: "invalid sub id",
		},
		{
			name:        "Error. Not found",
			reqRawSubID: "1",
			wantErr:     true,
			wantErrCode: "sub not found",
		},
	}

	for _, test := range tests {
		testContainers.Reset(t, ctx)
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			err := service.Delete(ctx, test.reqRawSubID)

			if test.wantErr {
				require.Error(t, err)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			if test.validate != nil {
				test.validate()
			}
		})
	}
}

func TestSubService_List(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	startDate := newDate(time.Now())
	endDate := startDate.AddMonths(1)

	tests := []struct {
		name        string
		prepare     func()
		rawLimit    string
		rawPage     string
		wantErr     bool
		wantErrCode string
		validate    func(got *ListResponse)
	}{
		{
			name: "Success. Listed 1",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			rawLimit: "5",
			rawPage:  "1",
			validate: func(got *ListResponse) {
				require.Equal(t, 3, got.TotalCount)
				require.Equal(t, 3, got.PageSize)
				require.Equal(t, 1, got.PageNumber)

				require.Len(t, got.Content, 3)
			},
		},
		{
			name: "Success. Listed 2",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			rawLimit: "2",
			rawPage:  "2",
			validate: func(got *ListResponse) {
				require.Equal(t, 3, got.TotalCount)
				require.Equal(t, 1, got.PageSize)
				require.Equal(t, 2, got.PageNumber)

				require.Len(t, got.Content, 1)
				require.Equal(t, int64(3), got.Content[0].ID)
			},
		},
		{
			name: "Success. Listed 3",
			prepare: func() {
				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})

				createSub(t, &CreateSubRequest{
					ServiceName: "yandex",
					Price:       400,
					UserID:      uuid.New(),
					StartDate:   startDate,
					EndDate:     &endDate,
				})
			},
			validate: func(got *ListResponse) {
				require.Equal(t, 3, got.TotalCount)
				require.Equal(t, 1, got.PageSize)
				require.Equal(t, 1, got.PageNumber)

				require.Len(t, got.Content, 1)
				require.Equal(t, int64(1), got.Content[0].ID)
			},
		},
		{
			name: "Success. Listed 3",
			validate: func(got *ListResponse) {
				require.Equal(t, 0, got.TotalCount)
				require.Equal(t, 0, got.PageSize)
				require.Equal(t, 1, got.PageNumber)

				require.NotNil(t, got.Content)
			},
		},
	}

	for _, test := range tests {
		testContainers.Reset(t, ctx)
		t.Run(test.name, func(t *testing.T) {
			if test.prepare != nil {
				test.prepare()
			}

			got, err := service.List(ctx, test.rawLimit, test.rawPage)

			if test.wantErr {
				require.Error(t, err)
				require.NotNil(t, got)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			if test.validate != nil {
				test.validate(got)
			}
		})
	}
}

func TestSubService_TotalAmountV2(t *testing.T) {
	ctx := context.Background()
	testContainers.Reset(t, ctx)

	repo := repository.NewSubsRepo(ctx, DB())
	logger := slog.Default()
	service := NewSub(repo, logger)

	userID := uuid.New()

	allSubs := []CreateSubRequest{}
	req1 := &CreateSubRequest{
		ServiceName: "yandex",
		Price:       400,
		UserID:      userID,
		StartDate:   newDate(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:     new(newDate(time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC))),
	}
	createSub(t, req1)
	allSubs = append(allSubs, *req1)

	req2 := &CreateSubRequest{
		ServiceName: "kion",
		Price:       800,
		UserID:      userID,
		StartDate:   newDate(time.Date(2000, 3, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:     new(newDate(time.Date(2000, 4, 1, 0, 0, 0, 0, time.UTC))),
	}
	createSub(t, req2)
	allSubs = append(allSubs, *req2)

	req3 := &CreateSubRequest{
		ServiceName: "kion",
		Price:       800,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Date(2000, 3, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:     new(newDate(time.Date(2000, 8, 1, 0, 0, 0, 0, time.UTC))),
	}
	createSub(t, req3)
	allSubs = append(allSubs, *req3)

	req4 := &CreateSubRequest{
		ServiceName: "kinopoisk",
		Price:       700,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:     new(newDate(time.Date(2001, 5, 1, 0, 0, 0, 0, time.UTC))),
	}
	createSub(t, req4)
	allSubs = append(allSubs, *req4)

	req5 := &CreateSubRequest{
		ServiceName: "netflix",
		Price:       700,
		UserID:      uuid.New(),
		StartDate:   newDate(time.Date(2000, 8, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:     new(newDate(time.Date(2000, 10, 2, 0, 0, 0, 0, time.UTC))),
	}
	createSub(t, req5)
	allSubs = append(allSubs, *req5)

	tests := []struct {
		name        string
		req         *TotalAmountRequest
		wantErr     bool
		wantErrCode string
		expectedSum int
	}{
		{
			name: "Success. With Only Service Name Filter",
			req: &TotalAmountRequest{
				RawSubname: "kion",
			},
			expectedSum: 4800,
		},
		{
			name: "Success. With Only UserID Filter",
			req: &TotalAmountRequest{
				RawUserID: userID.String(),
			},
			expectedSum: 2400,
		},
		{
			name: "Success. With Only From Filter",
			req: &TotalAmountRequest{
				RawFrom: "2000-03-01",
			},
			expectedSum: 16100,
		},
		{
			name: "Success. With Only To Filter",
			req: &TotalAmountRequest{
				RawTo: "2000-10-02",
			},
			expectedSum: 12700,
		},
		{
			name: "Success. With Both Date Filters",
			req: &TotalAmountRequest{
				RawFrom: "2000-05-05",
				RawTo:   "2000-08-09",
			},
			expectedSum: 4400,
		},
		{
			name: "Success. Multi Filters 1",
			req: &TotalAmountRequest{
				RawUserID:  userID.String(),
				RawSubname: "kion",
			},
			expectedSum: 800,
		},
		{
			name: "Success. Multi Filters 2",
			req: &TotalAmountRequest{
				RawUserID: uuid.NewString(),
				RawTo:     "1999-12-31",
			},
			expectedSum: 0,
		},
		{
			name: "Success. Multi Filters 3",
			req: &TotalAmountRequest{
				RawSubname: "kion",
				RawFrom:    "2000-04-01",
			},
			expectedSum: 3200,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got, err := service.TotalAmount(ctx, test.req)

			if test.wantErr {
				require.Error(t, err)
				require.NotNil(t, got)
				if test.wantErrCode != "" {
					require.Contains(t, err.Error(), test.wantErrCode)
				}
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectedSum, got.Sum)
		})
	}
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
	`, req.ServiceName, req.Price, req.UserID, req.StartDate, req.EndDate)
	require.NoError(t, err)
}

func countSubs(t *testing.T) int {
	var count int
	require.NoError(t, DB().GetContext(t.Context(), &count, `SELECT COUNT(*) FROM subs`))
	return count
}
