-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `block`
    CHANGE COLUMN `transaction_hashes` `transaction_hashes` MEDIUMTEXT;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
    DROP INDEX Idx_License;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim`
    CHANGE COLUMN `license` `license` TEXT;
-- +migrate StatementEnd
