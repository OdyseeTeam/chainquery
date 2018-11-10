-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim
  ADD COLUMN is_cert_valid TINYINT(1) NOT NULL,
  ADD COLUMN is_cert_processed TINYINT(1) NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  ADD INDEX idx_cert_valid (is_cert_valid),
  ADD INDEX idx_cert_processed (is_cert_processed);
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE claim
LEFT JOIN claim channel ON channel.claim_id = claim.publisher_id
SET claim.is_cert_processed = TRUE
WHERE channel.id IS NULL
AND claim.is_cert_processed = FALSE
-- +migrate StatementEnd
