-- +migrate Up

-- +migrate StatementBegin
DROP TABLE peer_claim_checkpoint;
-- +migrate StatementEnd

-- +migrate StatementBegin
DROP TABLE claim_checkpoint;
-- +migrate StatementEnd

-- +migrate StatementBegin
DROP TABLE peer_claim;
-- +migrate StatementEnd

-- +migrate StatementBegin
DROP TABLE peer;
-- +migrate StatementEnd

-- +migrate StatementBegin
DROP TABLE input_address;
-- +migrate StatementEnd

-- +migrate StatementBegin
DROP TABLE output_address;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE address
  DROP COLUMN tag,
  DROP COLUMN tag_url,
  CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE output
    DROP COLUMN hash160,
  CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction
    DROP COLUMN value,
  CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction_address
    DROP COLUMN latest_transaction_time;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE block
    DROP COLUMN median_time,
  DROP COLUMN target,
CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE unknown_claim
RENAME TO  abnormal_claim,
  ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  DROP FOREIGN KEY claim_ibfk_1;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  CHANGE COLUMN transaction_by_hash_id transaction_hash_id VARCHAR(70) CHARACTER SET 'latin1' COLLATE 'latin1_general_ci' NULL DEFAULT NULL ,
  CHANGE COLUMN is_n_s_f_w is_nsfw TINYINT(1) NOT NULL DEFAULT '0' ,
  CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  ADD CONSTRAINT claim_ibfk_1
FOREIGN KEY (transaction_hash_id)
REFERENCES transaction (hash)
  ON DELETE CASCADE
  ON UPDATE NO ACTION;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction
  DROP FOREIGN KEY transaction_ibfk_1;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction
  CHANGE COLUMN block_by_hash_id block_hash_id VARCHAR(70) CHARACTER SET 'latin1' COLLATE 'latin1_general_ci' NULL DEFAULT NULL ;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction
  ADD CONSTRAINT transaction_ibfk_1
FOREIGN KEY (block_hash_id)
REFERENCES block (hash)
  ON DELETE CASCADE
  ON UPDATE NO ACTION;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE support
  CHANGE COLUMN created created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CHANGE COLUMN modified modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd