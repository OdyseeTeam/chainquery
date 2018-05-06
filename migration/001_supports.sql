-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE support ADD COLUMN `created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE support ADD COLUMN `modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `job_status`
(
  `job_name`      VARCHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `last_sync`     DATETIME NOT NULL DEFAULT '0001-01-01 00:00:00',
  `is_success`    TINYINT(1) DEFAULT 0 NOT NULL,
  `error_message` TEXT,

  PRIMARY KEY `pk_jobstatus` (`job_name`)

)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `output_claim`
(
  `output_id` VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci COMMENT 'transaction hash of output',
  `vout_id` INTEGER UNSIGNED NOT NULL,
  `claim_id` CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `type` VARCHAR(30) DEFAULT 'create',

  PRIMARY KEY `pk_output_claim` (`output_id`,`vout_id`,`claim_id`),
  FOREIGN KEY `fk_output` (`output_id`,`vout_id`) REFERENCES `output` (`transaction_hash`,`vout`) ON DELETE CASCADE ON UPDATE NO ACTION,
  FOREIGN KEY `fk_claim` (`claim_id`) REFERENCES `claim` (`claim_id`) ON DELETE CASCADE ON UPDATE NO ACTION
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd