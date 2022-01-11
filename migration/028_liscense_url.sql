-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim CHANGE COLUMN license_url license_url VARCHAR(255) CHARACTER SET 'utf8' COLLATE 'utf8_unicode_ci' NULL DEFAULT NULL;
-- +migrate StatementEnd