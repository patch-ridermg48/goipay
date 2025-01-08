-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX unique_tx_id_idx ON invoices (tx_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX unique_tx_id_idx;
-- +goose StatementEnd