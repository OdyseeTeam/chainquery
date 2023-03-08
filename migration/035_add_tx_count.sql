-- +migrate Up
-- +migrate StatementBegin
alter table block add tx_count int not null after version_hex;
-- +migrate StatementEnd