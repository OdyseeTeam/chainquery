-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_Height` (`height` ASC);
-- +migrate StatementEnd