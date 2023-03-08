-- +migrate Up
-- +migrate StatementBegin
alter table claim drop column website_url;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column country;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column state;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column city;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column `code`;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column latitude;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column longitude;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column license_url;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column preview;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim drop column os;
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim modify fee_address varchar(40) collate latin1_general_ci null;
-- +migrate StatementEnd
-- +migrate StatementBegin
update claim set fee_address = null where fee_address = '';
-- +migrate StatementEnd
-- +migrate StatementBegin
alter table claim modify version varchar(10) collate latin1_general_ci null;
-- +migrate StatementEnd
-- +migrate StatementBegin
update claim set version = null where version = '';
-- +migrate StatementEnd