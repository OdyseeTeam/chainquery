-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim ADD INDEX (sd_hash ASC);
-- +migrate StatementEnd