-- +migrate Up
-- +migrate StatementBegin
ALTER TABLE claim ADD COLUMN claim_count BIGINT NOT NULL DEFAULT 0;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim ADD INDEX (claim_count);
-- +migrate StatementEnd