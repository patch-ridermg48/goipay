-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS bnb_crypto_data(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_pub_key TEXT NOT NULL UNIQUE,
    last_major_index INTEGER NOT NULL DEFAULT 0,
    last_minor_index INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE crypto_data ADD COLUMN bnb_id UUID REFERENCES bnb_crypto_data (id);

ALTER TYPE coin_type ADD VALUE 'BNB';
--BEP20
ALTER TYPE coin_type ADD VALUE 'BSC-USD_BEP20';
ALTER TYPE coin_type ADD VALUE 'USDC_BEP20';
ALTER TYPE coin_type ADD VALUE 'DAI_BEP20';
ALTER TYPE coin_type ADD VALUE 'BUSD_BEP20';
ALTER TYPE coin_type ADD VALUE 'WBTC_BEP20';
ALTER TYPE coin_type ADD VALUE 'BTCB_BEP20';
ALTER TYPE coin_type ADD VALUE 'UNI_BEP20';
ALTER TYPE coin_type ADD VALUE 'LINK_BEP20';
ALTER TYPE coin_type ADD VALUE 'AAVE_BEP20';
ALTER TYPE coin_type ADD VALUE 'MATIC_BEP20';
ALTER TYPE coin_type ADD VALUE 'SHIB_BEP20';
ALTER TYPE coin_type ADD VALUE 'ATOM_BEP20';
ALTER TYPE coin_type ADD VALUE 'ARB_BEP20';
ALTER TYPE coin_type ADD VALUE 'ETH_BEP20';
ALTER TYPE coin_type ADD VALUE 'XRP_BEP20';
ALTER TYPE coin_type ADD VALUE 'ADA_BEP20';
ALTER TYPE coin_type ADD VALUE 'TRX_BEP20';
ALTER TYPE coin_type ADD VALUE 'DOGE_BEP20';
ALTER TYPE coin_type ADD VALUE 'LTC_BEP20';
ALTER TYPE coin_type ADD VALUE 'BCH_BEP20';
ALTER TYPE coin_type ADD VALUE 'TWT_BEP20';
ALTER TYPE coin_type ADD VALUE 'AVAX_BEP20';
ALTER TYPE coin_type ADD VALUE 'CAKE_BEP20';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crypto_data DROP COLUMN bnb_id;

DROP TABLE bnb_crypto_data;
-- +goose StatementEnd