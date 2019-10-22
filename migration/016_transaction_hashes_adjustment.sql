-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `block`
    CHANGE COLUMN `transaction_hashes` `transaction_hashes` MEDIUMTEXT;
-- +migrate StatementEnd
