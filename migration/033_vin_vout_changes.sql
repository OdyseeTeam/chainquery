-- +migrate Up
-- +migrate StatementBegin
alter table input drop column prevout_spend_updated;
-- +migrate StatementEnd
