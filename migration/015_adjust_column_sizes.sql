-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `tag`
    CHANGE COLUMN `tag` `tag` VARCHAR(500) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE address CHANGE COLUMN address address VARCHAR(50) CHARACTER SET latin1 COLLATE latin1_general_ci UNIQUE NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `input` ADD COLUMN `witness` TEXT;
-- +migrate StatementEnd

