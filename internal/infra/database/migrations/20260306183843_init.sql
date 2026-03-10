-- +goose Up
-- +goose StatementBegin
CREATE TABLE subs (
    id              BIGSERIAL PRIMARY KEY,
    service_name    TEXT NOT NULL,
    price           INTEGER NOT NULL,
    user_id         UUID NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, service_name, start_date)
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_subs_set_updated_at
BEFORE UPDATE ON subs
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE subs;
-- +goose StatementEnd
