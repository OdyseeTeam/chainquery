-- +migrate Up
-- +migrate StatementBegin
drop index Idx_outpoint on support;
-- +migrate StatementEnd

-- +migrate StatementBegin
create unique index Idx_uniq_outpoint on support (transaction_hash_id, vout);
-- +migrate StatementEnd
