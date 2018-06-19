
CREATE TABLE `address`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `address` varchar(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `first_seen` datetime DEFAULT NULL,
  `tag` varchar(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tag_url` varchar(200) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `address` (`address`),
  UNIQUE KEY `Idx_AddressAddress` (`address`),
  UNIQUE KEY `tag` (`tag`),
  UNIQUE KEY `Idx_AddressTag` (`tag`),
  KEY `Idx_AddressCreated` (`created`),
  KEY `Idx_AddressModified` (`modified`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `application_status`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `app_version` int(11) NOT NULL,
  `data_version` int(11) NOT NULL,
  `api_version` int(11) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `block`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `bits` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL,
  `chainwork` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `confirmations` int(10) unsigned NOT NULL,
  `difficulty` double(18,8) NOT NULL,
  `hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `height` bigint(20) unsigned NOT NULL,
  `median_time` bigint(20) unsigned NOT NULL,
  `merkle_root` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `name_claim_root` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `nonce` bigint(20) unsigned NOT NULL,
  `previous_block_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `next_block_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `block_size` bigint(20) unsigned NOT NULL,
  `target` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `block_time` bigint(20) unsigned NOT NULL,
  `version` bigint(20) unsigned NOT NULL,
  `version_hex` varchar(10) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `transaction_hashes` text COLLATE utf8mb4_unicode_ci,
  `transactions_processed` tinyint(1) NOT NULL DEFAULT '0',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `Idx_BlockHash` (`hash`),
  KEY `Idx_BlockHeight` (`height`),
  KEY `Idx_BlockTime` (`block_time`),
  KEY `Idx_MedianTime` (`median_time`),
  KEY `Idx_PreviousBlockHash` (`previous_block_hash`),
  KEY `Idx_BlockCreated` (`created`),
  KEY `Idx_BlockModified` (`modified`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `claim`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `transaction_by_hash_id` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `vout` int(10) unsigned NOT NULL,
  `name` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  `claim_id` char(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `claim_type` tinyint(2) NOT NULL,
  `publisher_id` char(40) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL COMMENT 'references a ClaimId with CertificateType',
  `publisher_sig` varchar(200) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `certificate` text COLLATE utf8mb4_unicode_ci,
  `sd_hash` varchar(120) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `transaction_time` bigint(20) unsigned DEFAULT NULL,
  `version` varchar(10) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `value_as_hex` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `value_as_json` mediumtext COLLATE utf8mb4_unicode_ci,
  `valid_at_height` int(10) unsigned NOT NULL,
  `height` int(10) unsigned NOT NULL,
  `effective_amount` bigint(20) unsigned NOT NULL DEFAULT '0',
  `author` varchar(512) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `description` mediumtext COLLATE utf8mb4_unicode_ci,
  `content_type` varchar(162) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `is_n_s_f_w` tinyint(1) NOT NULL DEFAULT '0',
  `language` varchar(20) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `thumbnail_url` text COLLATE utf8mb4_unicode_ci,
  `title` text COLLATE utf8mb4_unicode_ci,
  `fee` double(58,8) NOT NULL DEFAULT '0.00000000',
  `fee_currency` char(30) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `is_filtered` tinyint(1) NOT NULL DEFAULT '0',
  `bid_state` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Accepted',
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `fee_address` varchar(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `claim_address` varchar(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `Idx_ClaimUnique` (`transaction_by_hash_id`,`vout`,`claim_id`),
  KEY `FK_ClaimPublisher` (`publisher_id`),
  KEY `Idx_Claim` (`claim_id`),
  KEY `Idx_ClaimTransactionTime` (`transaction_time`),
  KEY `Idx_ClaimCreated` (`created`),
  KEY `Idx_ClaimModified` (`modified`),
  KEY `Idx_ClaimValidAtHeight` (`valid_at_height`),
  KEY `Idx_ClaimBidState` (`bid_state`),
  KEY `Idx_ClaimName` (`name`(255)),
  KEY `Idx_ClaimAuthor` (`author`(191)),
  KEY `Idx_ClaimContentType` (`content_type`),
  KEY `Idx_ClaimLanguage` (`language`),
  KEY `Idx_ClaimTitle` (`title`(191)),
  KEY `Idx_FeeAddress` (`fee_address`),
  KEY `Idx_ClaimAddress` (`claim_address`),
  KEY `Idx_ClaimOutpoint` (`transaction_by_hash_id`,`vout`),
  CONSTRAINT `claim_ibfk_1` FOREIGN KEY (`transaction_by_hash_id`) REFERENCES `transaction` (`hash`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `claim_checkpoint`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `claim_id` char(40) COLLATE latin1_general_ci NOT NULL,
  `checkpoint` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `is_available` tinyint(1) NOT NULL,
  `head_available` tinyint(1) NOT NULL,
  `s_d_available` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `idx_claim_id` (`claim_id`),
  KEY `idx_checkpoint` (`checkpoint`),
  KEY `idx_is_available` (`is_available`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

CREATE TABLE `gorp_migrations`
(
  `id` varchar(255) NOT NULL,
  `applied_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `input`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `transaction_id` bigint(20) unsigned NOT NULL,
  `transaction_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `input_address_id` bigint(20) unsigned DEFAULT NULL,
  `is_coinbase` tinyint(1) NOT NULL DEFAULT '0',
  `coinbase` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `prevout_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `prevout_n` int(10) unsigned DEFAULT NULL,
  `prevout_spend_updated` tinyint(1) NOT NULL DEFAULT '0',
  `sequence` int(10) unsigned NOT NULL,
  `value` double(18,8) DEFAULT NULL,
  `script_sig_asm` text CHARACTER SET latin1 COLLATE latin1_general_ci,
  `script_sig_hex` text CHARACTER SET latin1 COLLATE latin1_general_ci,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `FK_InputAddress` (`input_address_id`),
  KEY `FK_InputTransaction` (`transaction_id`),
  KEY `Idx_InputValue` (`value`),
  KEY `Idx_PrevoutHash` (`prevout_hash`),
  KEY `Idx_InputCreated` (`created`),
  KEY `Idx_InputModified` (`modified`),
  KEY `Idx_InputTransactionHash` (`transaction_hash`),
  CONSTRAINT `input_ibfk_2` FOREIGN KEY (`transaction_id`) REFERENCES `transaction` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `input_address`
(
  `input_id` bigint(20) unsigned NOT NULL,
  `address_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`input_id`,`address_id`),
  KEY `Idx_InputsAddressesAddress` (`address_id`),
  CONSTRAINT `input_address_ibfk_1` FOREIGN KEY (`input_id`) REFERENCES `input` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT `input_address_ibfk_2` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `job_status`
(
  `job_name` varchar(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `last_sync` datetime NOT NULL DEFAULT '0001-01-01 00:00:00',
  `is_success` tinyint(1) NOT NULL DEFAULT '0',
  `error_message` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`job_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `output`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `transaction_id` bigint(20) unsigned NOT NULL,
  `transaction_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `value` double(18,8) DEFAULT NULL,
  `vout` int(10) unsigned NOT NULL,
  `type` varchar(20) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `script_pub_key_asm` text CHARACTER SET latin1 COLLATE latin1_general_ci,
  `script_pub_key_hex` text CHARACTER SET latin1 COLLATE latin1_general_ci,
  `required_signatures` int(10) unsigned DEFAULT NULL,
  `hash160` varchar(50) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `address_list` text CHARACTER SET latin1 COLLATE latin1_general_ci,
  `is_spent` tinyint(1) NOT NULL DEFAULT '0',
  `spent_by_input_id` bigint(20) unsigned DEFAULT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `claim_id` char(40) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `UK_Output` (`transaction_hash`,`vout`),
  KEY `FK_OutputTransaction` (`transaction_id`),
  KEY `FK_OutputSpentByInput` (`spent_by_input_id`),
  KEY `Idx_OutputValue` (`value`),
  KEY `Idx_Oupoint` (`vout`,`transaction_hash`) COMMENT 'needed for references in this column order',
  KEY `Idx_OuptutCreated` (`created`),
  KEY `Idx_OutputModified` (`modified`),
  KEY `fk_claim` (`claim_id`),
  KEY `Idx_IsSpent` (`is_spent`),
  KEY `Idx_SpentOutput` (`transaction_hash`,`vout`,`is_spent`),
  CONSTRAINT `output_ibfk_1` FOREIGN KEY (`transaction_id`) REFERENCES `transaction` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `output_address`
(
  `output_id` bigint(20) unsigned NOT NULL,
  `address_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`output_id`,`address_id`),
  KEY `Idx_OutputsAddressesAddress` (`address_id`),
  CONSTRAINT `output_address_ibfk_1` FOREIGN KEY (`output_id`) REFERENCES `output` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT `output_address_ibfk_2` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `peer`
(
  `node_id` char(100) COLLATE latin1_general_ci NOT NULL,
  `known_i_p_list` text COLLATE latin1_general_ci,
  PRIMARY KEY (`node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

CREATE TABLE `peer_claim`
(
  `peer_id` char(100) COLLATE latin1_general_ci NOT NULL,
  `claim_id` char(40) COLLATE latin1_general_ci NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_seen` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`peer_id`,`claim_id`),
  KEY `idx_claim_id` (`claim_id`),
  KEY `idx_created` (`created`),
  KEY `idx_last_seen` (`last_seen`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

CREATE TABLE `peer_claim_checkpoint`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `peer_id` char(100) COLLATE latin1_general_ci NOT NULL,
  `claim_id` char(40) COLLATE latin1_general_ci NOT NULL,
  `checkpoint` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `is_available` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `idx_peer_id` (`peer_id`),
  KEY `idx_claim_id` (`claim_id`),
  KEY `idx_checkpoint` (`checkpoint`),
  KEY `idx_is_available` (`is_available`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1 COLLATE=latin1_general_ci;

CREATE TABLE `support`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `supported_claim_id` char(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `support_amount` double(18,8) NOT NULL DEFAULT '0.00000000',
  `bid_state` varchar(20) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'Accepted',
  `transaction_hash_id` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `vout` int(10) unsigned NOT NULL,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `Idx_state` (`bid_state`),
  KEY `Idx_supportedclaimid` (`supported_claim_id`),
  KEY `Idx_transaction` (`transaction_hash_id`),
  KEY `Idx_vout` (`vout`),
  KEY `Idx_outpoint` (`transaction_hash_id`,`vout`),
  CONSTRAINT `fk_transaction` FOREIGN KEY (`transaction_hash_id`) REFERENCES `transaction` (`hash`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `transaction`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `block_by_hash_id` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `input_count` int(10) unsigned NOT NULL,
  `output_count` int(10) unsigned NOT NULL,
  `value` double(18,8) NOT NULL,
  `fee` double(18,8) NOT NULL DEFAULT '0.00000000',
  `transaction_time` bigint(20) unsigned DEFAULT NULL,
  `transaction_size` bigint(20) unsigned NOT NULL,
  `hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  `version` int(11) NOT NULL,
  `lock_time` int(10) unsigned NOT NULL,
  `raw` text COLLATE utf8mb4_unicode_ci,
  `created` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `Idx_TransactionHash` (`hash`),
  KEY `FK_TransactionBlockHash` (`block_by_hash_id`),
  KEY `Idx_TransactionTime` (`transaction_time`),
  KEY `Idx_TransactionCreatedTime` (`created_time`),
  KEY `Idx_TransactionCreated` (`created`),
  KEY `Idx_TransactionModified` (`modified`),
  CONSTRAINT `transaction_ibfk_1` FOREIGN KEY (`block_by_hash_id`) REFERENCES `block` (`hash`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `transaction_address`
(
  `transaction_id` bigint(20) unsigned NOT NULL,
  `address_id` bigint(20) unsigned NOT NULL,
  `debit_amount` double(18,8) NOT NULL DEFAULT '0.00000000' COMMENT 'Sum of the inputs to this address for the tx',
  `credit_amount` double(18,8) NOT NULL DEFAULT '0.00000000' COMMENT 'Sum of the outputs to this address for the tx',
  `latest_transaction_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`transaction_id`,`address_id`),
  KEY `Idx_TransactionsAddressesAddress` (`address_id`),
  KEY `Idx_TransactionsAddressesLatestTransactionTime` (`latest_transaction_time`),
  KEY `Idx_TransactionsAddressesDebit` (`debit_amount`),
  KEY `Idx_TransactionsAddressesCredit` (`credit_amount`),
  CONSTRAINT `transaction_address_ibfk_1` FOREIGN KEY (`transaction_id`) REFERENCES `transaction` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT `transaction_address_ibfk_2` FOREIGN KEY (`address_id`) REFERENCES `address` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

CREATE TABLE `unknown_claim`
(
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  `claim_id` varchar(160) COLLATE utf8mb4_unicode_ci NOT NULL,
  `is_update` tinyint(1) NOT NULL DEFAULT '0',
  `block_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `transaction_hash` varchar(70) CHARACTER SET latin1 COLLATE latin1_general_ci DEFAULT NULL,
  `vout` int(10) unsigned NOT NULL,
  `output_id` bigint(20) unsigned NOT NULL,
  `value_as_hex` mediumtext COLLATE utf8mb4_unicode_ci NOT NULL,
  `value_as_json` mediumtext COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `id` (`id`),
  KEY `Idx_UnknowClaimBlockHash` (`block_hash`),
  KEY `Idx_UnknowClaimOutput` (`output_id`),
  KEY `Idx_UnknowClaimTxHash` (`transaction_hash`),
  CONSTRAINT `unknown_claim_ibfk_1` FOREIGN KEY (`output_id`) REFERENCES `output` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- Dump completed on 2018-06-18 19:54:48