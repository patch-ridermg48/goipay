-- +goose Up
-- +goose StatementBegin
ALTER TABLE invoices 
ADD CONSTRAINT unique_tx_id UNIQUE USING INDEX unique_tx_id_idx;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE invoices DROP CONSTRAINT unique_tx_id;
-- +goose StatementEnd