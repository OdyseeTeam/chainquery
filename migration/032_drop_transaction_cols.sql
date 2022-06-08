-- +migrate Up
-- +migrate StatementBegin
alter table transaction drop column fee;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table transaction drop column raw;
-- +migrate StatementEnd