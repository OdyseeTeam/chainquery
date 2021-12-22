-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE support ADD COLUMN supported_by_claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' DEFAULT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE support ADD INDEX idx_supported_by (supported_by_claim_id);
-- +migrate StatementEnd
