-- +migrate Up

-- +migrate StatementBegin
alter table claim modify value_as_json JSON null;
-- +migrate StatementEnd