-- +goose Up
-- +goose StatementBegin
CREATE TABLE subs (
    id              BIGSERIAL PRIMARY KEY,
    service_name    TEXT NOT NULL,
    price           INTEGER NOT NULL,
    user_id         UUID NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE,
    UNIQUE(user_id, service_name, start_date)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE subs;
-- +goose StatementEnd
