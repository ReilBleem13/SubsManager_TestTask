package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ReilBleem13/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type SubcRepo interface {
	Create(ctx context.Context, req *domain.Sub) error
	Get(ctx context.Context, id int64) (*domain.Sub, error)
	Update(ctx context.Context, req *SubUpdate) (*domain.Sub, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]domain.Sub, int, error)
	TotalAmount(ctx context.Context, filter *SubFilter) (int, error)
}

type subs struct {
	db *sqlx.DB
}

func NewSubsRepo(ctx context.Context, dbClient *sqlx.DB) SubcRepo {
	return &subs{
		db: dbClient,
	}
}

func (s *subs) Create(ctx context.Context, req *domain.Sub) error {
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO subs (
			service_name,
			price,
			user_id,
			start_date,
			end_date
		) VALUES($1, $2, $3, $4, $5)
		 RETURNING id
	`,
		req.ServiceName, req.Price, req.UserID, req.StartDate, req.EndDate,
	).Scan(&req.ID)

	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExist().WithMessage("Sub already exists")
		}
		return err
	}
	return nil
}

func (s *subs) Get(ctx context.Context, id int64) (*domain.Sub, error) {
	var sub domain.Sub

	if err := s.db.GetContext(ctx, &sub, `
		SELECT
			id,
			service_name,
			price,
			user_id,
			start_date,
			end_date
		FROM subs
		WHERE id = $1
	`, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound().WithMessage("Sub not found")
		}
		return nil, err
	}
	return &sub, nil
}

func (s *subs) Update(ctx context.Context, req *SubUpdate) (*domain.Sub, error) {
	var updatedSub domain.Sub

	query, args, err := sqlx.Named(`
		UPDATE subs SET
			service_name = COALESCE(:service_name, service_name),
			price = COALESCE(:price, price),
			user_id = COALESCE(:user_id, user_id),
			start_date = COALESCE(:start_date, start_date),
			end_date = COALESCE(:end_date, end_date)
		WHERE id = :id
		RETURNING *
	`, req)
	if err != nil {
		return nil, err
	}

	query = s.db.Rebind(query)

	err = s.db.GetContext(ctx, &updatedSub, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound().WithMessage("Sub not found")
		}
		return nil, err
	}
	return &updatedSub, nil
}

func (s *subs) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM subs
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return domain.ErrNotFound().WithMessage("Sub not found")
	}
	return nil
}

type subRow struct {
	domain.Sub
	TotalCount int `db:"total_count"`
}

func (s *subs) List(ctx context.Context, limit, offset int) ([]domain.Sub, int, error) {
	query := `
		SELECT
			id,
			service_name,
			price,
			user_id,
			start_date,
			end_date,
			COUNT(*) OVER() AS total_count
		FROM subs
		ORDER BY id 
		LIMIT $1 OFFSET $2
	`

	var rows []subRow
	if err := s.db.SelectContext(ctx, &rows, query,
		limit,
		offset,
	); err != nil {
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []domain.Sub{}, 0, nil
	}

	subs := make([]domain.Sub, len(rows))
	for i := range rows {
		subs[i] = rows[i].Sub
	}
	return subs, rows[0].TotalCount, nil
}

func (s *subs) TotalAmount(ctx context.Context, filter *SubFilter) (int, error) {
	query := `
		SELECT
			COALESCE(SUM(price), 0)
		FROM subs
	`

	conds, args := s.handleFilter(filter)

	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	var totalSum int
	if err := s.db.GetContext(ctx, &totalSum, query, args...); err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	return totalSum, nil
}

func (s *subs) handleFilter(filter *SubFilter) ([]string, []interface{}) {
	var (
		conds []string
		args  []interface{}
	)

	if filter != nil {
		if filter.ServiceName != "" {
			conds = append(conds, fmt.Sprintf("service_name = $%d", len(args)+1))
			args = append(args, filter.ServiceName)
		}

		if filter.UserID != nil {
			conds = append(conds, fmt.Sprintf("user_id = $%d", len(args)+1))
			args = append(args, filter.UserID)
		}

		if filter.From != nil {
			conds = append(conds, fmt.Sprintf("start_date >= $%d", len(args)+1))
			args = append(args, filter.From)
		}

		if filter.To != nil {
			conds = append(conds, fmt.Sprintf("start_date <= $%d", len(args)+1))
			args = append(args, filter.To)
		}
	}
	return conds, args
}
