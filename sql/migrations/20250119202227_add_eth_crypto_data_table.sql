-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS eth_crypto_data(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_pub_key TEXT NOT NULL UNIQUE,
    last_major_index INTEGER NOT NULL DEFAULT 0,
    last_minor_index INTEGER NOT NULL DEFAULT 0
);

ALTER TABLE crypto_data ADD COLUMN eth_id UUID REFERENCES eth_crypto_data (id);

ALTER TYPE coin_type ADD VALUE 'USDT_ERC20';
ALTER TYPE coin_type ADD VALUE 'USDC_ERC20';
ALTER TYPE coin_type ADD VALUE 'DAI_ERC20';
ALTER TYPE coin_type ADD VALUE 'WBTC_ERC20';
ALTER TYPE coin_type ADD VALUE 'UNI_ERC20';
ALTER TYPE coin_type ADD VALUE 'LINK_ERC20';
ALTER TYPE coin_type ADD VALUE 'AAVE_ERC20';
ALTER TYPE coin_type ADD VALUE 'CRV_ERC20';
ALTER TYPE coin_type ADD VALUE 'MATIC_ERC20';
ALTER TYPE coin_type ADD VALUE 'SHIB_ERC20';
ALTER TYPE coin_type ADD VALUE 'BNB_ERC20';
ALTER TYPE coin_type ADD VALUE 'ATOM_ERC20';
ALTER TYPE coin_type ADD VALUE 'ARB_ERC20';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE crypto_data DROP COLUMN eth_id;

DROP TABLE eth_crypto_data;
-- +goose StatementEnd