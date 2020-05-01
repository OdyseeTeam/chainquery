-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `chainquery`.`claim`
    ADD INDEX `Idx_sdhash` (`sd_hash` ASC);
-- +migrate StatementEnd