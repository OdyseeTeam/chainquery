-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE input
    ADD INDEX Idx_TxHashVin (transaction_hash ASC, vin ASC);
-- +migrate StatementEnd