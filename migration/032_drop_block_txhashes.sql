-- +migrate Up
-- +migrate StatementBegin
alter table block drop column transaction_hashes;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table block drop column transactions_processed;
-- +migrate StatementEnd
