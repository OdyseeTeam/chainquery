-- +migrate Up
-- +migrate StatementBegin
alter table block add processing_state varchar(20) null after tx_count;
-- +migrate StatementEnd

