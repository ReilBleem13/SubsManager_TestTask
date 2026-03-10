package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Date time.Time

func (d Date) AddMonths(n int) Date {
	t := time.Time(d).AddDate(0, n, 0)
	return Date(t)
}

func (d Date) Before(other Date) bool {
	return time.Time(d).Before(time.Time(other))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	*d = Date(t)
	return nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).Format("2006-01-02"))
}

func (d *Date) Scan(value interface{}) error {
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("cannot scan %T into Date", value)
	}

	y, m, day := t.Date()
	*d = Date(time.Date(y, m, day, 0, 0, 0, 0, time.UTC))
	return nil
}

func (d Date) Value() (driver.Value, error) {
	return time.Time(d), nil
}

type Sub struct {
	ID          int64     `json:"id" db:"id"`
	ServiceName string    `json:"service_name" db:"service_name"`
	Price       int       `json:"price" db:"price"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	StartDate   Date      `json:"start_date" db:"start_date"`
	EndDate     Date      `json:"end_date" db:"end_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
