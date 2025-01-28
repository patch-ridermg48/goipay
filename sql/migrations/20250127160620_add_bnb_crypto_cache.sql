-- +goose Up
-- +goose StatementBegin
INSERT INTO crypto_cache(coin) VALUES ('BNB');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM crypto_cache WHERE coin = 'BNB';
-- +goose StatementEnd
