-- +migrate Up
-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `claim_checkpoint`
(
  `id` SERIAL,
  `claim_id` CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `checkpoint` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `is_available` TINYINT(1) NOT NULL,
  `head_available` TINYINT(1) NOT NULL,
  `s_d_available` TINYINT(1) NOT NULL,

  PRIMARY KEY `pk_claim_checkpoint` (`id`),
  INDEX `idx_claim_id` (`claim_id`),
  INDEX `idx_checkpoint` (`checkpoint`),
  INDEX `idx_is_available` (`is_available`)
);
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `peer`
(
  `node_id` CHAR(100) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `known_i_p_list` TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,

  PRIMARY KEY `pk_id` (`node_id`),
  CONSTRAINT `cnt_known_i_p_list` CHECK(`known_i_p_list` IS NULL OR JSON_VALID(`known_i_p_list`))
);
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `peer_claim`
(
  `peer_id` CHAR(100) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `claim_id` CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_seen` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY `pk_peer_claim` (`peer_id`,`claim_id`),
  INDEX `idx_claim_id` (`claim_id`),
  INDEX `idx_created` (`created`),
  INDEX `idx_last_seen` (`last_seen`)

);
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS `peer_claim_checkpoint`
(
  `id` SERIAL,
  `peer_id` CHAR(100) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `claim_id` CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `checkpoint` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `is_available` TINYINT(1) NOT NULL,

  PRIMARY KEY `pk_peer_claim_checkpoint` (`id`),
  INDEX `idx_peer_id` (`peer_id`),
  INDEX `idx_claim_id` (`claim_id`),
  INDEX `idx_checkpoint` (`checkpoint`),
  INDEX `idx_is_available` (`is_available`)

);
-- +migrate StatementEnd


