-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS ltc_crypto_data(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_pub_key TEXT NOT NULL UNIQUE,
    last_major_index INTEGER NOT NULL DEFAULT 0,
    last_minor_index INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE crypto_data ADD COLUMN ltc_id UUID REFERENCES ltc_crypto_data (id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crypto_data DROP COLUMN ltc_id;

DROP TABLE ltc_crypto_data;
-- +goose StatementEnd