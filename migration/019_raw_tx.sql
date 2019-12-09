-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE transaction
    CHANGE COLUMN raw raw MEDIUMTEXT NULL DEFAULT NULL;
-- +migrate StatementEnd