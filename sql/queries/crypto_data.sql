-- name: CreateCryptoData :one
INSERT INTO crypto_data(xmr_id, btc_id, ltc_id, eth_id, user_id) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindCryptoDataByUserId :one
SELECT * FROM crypto_data 
WHERE user_id = $1;

-- name: SetXMRCryptoDataByUserId :one
UPDATE crypto_data
SET xmr_id = $2 
WHERE user_id = $1
RETURNING *;

-- name: SetBTCCryptoDataByUserId :one
UPDATE crypto_data
SET btc_id = $2
WHERE user_id = $1
RETURNING *;

-- name: SetLTCCryptoDataByUserId :one
UPDATE crypto_data
SET ltc_id = $2
WHERE user_id = $1
RETURNING *;

-- name: SetETHCryptoDataByUserId :one
UPDATE crypto_data
SET eth_id = $2
WHERE user_id = $1
RETURNING *;


-- XMR
-- name: CreateXMRCryptoData :one
INSERT INTO xmr_crypto_data(priv_view_key, pub_spend_key) VALUES ($1, $2)
RETURNING *;

-- name: FindKeysAndLockXMRCryptoDataById :one
SELECT priv_view_key, pub_spend_key
FROM xmr_crypto_data
WHERE id = $1
FOR SHARE;

-- name: UpdateKeysXMRCryptoDataById :one
UPDATE xmr_crypto_data
SET priv_view_key = $2,
    pub_spend_key = $3,
    last_major_index = 0,
    last_minor_index = 0
WHERE id = $1
RETURNING *;

-- name: FindIndicesAndLockXMRCryptoDataById :one
SELECT last_major_index, last_minor_index 
FROM xmr_crypto_data
WHERE id = $1
FOR UPDATE;

-- name: UpdateIndicesXMRCryptoDataById :one
UPDATE xmr_crypto_data
SET last_major_index = $2,
    last_minor_index = $3
WHERE id = $1
RETURNING *;

-- BTC
-- name: CreateBTCCryptoData :one
INSERT INTO btc_crypto_data(master_pub_key) VALUES ($1)
RETURNING *;

-- name: FindKeysAndLockBTCCryptoDataById :one
SELECT master_pub_key
FROM btc_crypto_data
WHERE id = $1
FOR SHARE;

-- name: UpdateKeysBTCCryptoDataById :one
UPDATE btc_crypto_data
SET master_pub_key = $2,
    last_major_index = 0,
    last_minor_index = 0
WHERE id = $1
RETURNING *;

-- name: FindIndicesAndLockBTCCryptoDataById :one
SELECT last_major_index, last_minor_index 
FROM btc_crypto_data
WHERE id = $1
FOR UPDATE;

-- name: UpdateIndicesBTCCryptoDataById :one
UPDATE btc_crypto_data
SET last_major_index = $2,
    last_minor_index = $3
WHERE id = $1
RETURNING *;

-- LTC
-- name: CreateLTCCryptoData :one
INSERT INTO ltc_crypto_data(master_pub_key) VALUES ($1)
RETURNING *;

-- name: FindKeysAndLockLTCCryptoDataById :one
SELECT master_pub_key
FROM ltc_crypto_data
WHERE id = $1
FOR SHARE;

-- name: UpdateKeysLTCCryptoDataById :one
UPDATE ltc_crypto_data
SET master_pub_key = $2,
    last_major_index = 0,
    last_minor_index = 0
WHERE id = $1
RETURNING *;

-- name: FindIndicesAndLockLTCCryptoDataById :one
SELECT last_major_index, last_minor_index 
FROM ltc_crypto_data
WHERE id = $1
FOR UPDATE;

-- name: UpdateIndicesLTCCryptoDataById :one
UPDATE ltc_crypto_data
SET last_major_index = $2,
    last_minor_index = $3
WHERE id = $1
RETURNING *;

-- ETH
-- name: CreateETHCryptoData :one
INSERT INTO eth_crypto_data(master_pub_key) VALUES ($1)
RETURNING *;

-- name: FindKeysAndLockETHCryptoDataById :one
SELECT master_pub_key
FROM eth_crypto_data
WHERE id = $1
FOR SHARE;

-- name: UpdateKeysETHCryptoDataById :one
UPDATE eth_crypto_data
SET master_pub_key = $2,
    last_major_index = 0,
    last_minor_index = 0
WHERE id = $1
RETURNING *;

-- name: FindIndicesAndLockETHCryptoDataById :one
SELECT last_major_index, last_minor_index 
FROM eth_crypto_data
WHERE id = $1
FOR UPDATE;

-- name: UpdateIndicesETHCryptoDataById :one
UPDATE eth_crypto_data
SET last_major_index = $2,
    last_minor_index = $3
WHERE id = $1
RETURNING *;