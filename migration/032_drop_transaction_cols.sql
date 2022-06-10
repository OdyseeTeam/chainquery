-- +migrate Up
-- +migrate StatementBegin
alter table transaction drop column fee;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table transaction drop column raw;
-- +migrate StatementEnd
-- +migrate StatementBegin
drop trigger tg_insert_value;
-- +migrate StatementEnd
-- +migrate StatementBegin
drop trigger tg_update_value;
-- +migrate StatementEnd
-- +migrate StatementBegin
drop trigger tg_insert_balance;
-- +migrate StatementEnd