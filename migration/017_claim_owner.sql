-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim
    ADD COLUMN transaction_hash_update VARCHAR(70),
    ADD COLUMN vout_update INT(10) UNSIGNED;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
    ADD INDEX (transaction_hash_update,vout_update);
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE claim SET transaction_hash_update = transaction_hash_id, vout_update = vout;
-- +migrate StatementEnd

