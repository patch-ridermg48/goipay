-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS btc_crypto_data(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_pub_key TEXT NOT NULL UNIQUE,
    last_major_index INTEGER NOT NULL DEFAULT 0,
    last_minor_index INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE crypto_data ADD COLUMN btc_id UUID REFERENCES btc_crypto_data (id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crypto_data DROP COLUMN btc_id;

DROP TABLE btc_crypto_data;
-- +goose StatementEnd
