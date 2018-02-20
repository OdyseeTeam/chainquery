-- +migrate Up

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS blocks (
    bits VARCHAR(20) NOT NULL,
    chainwork VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    confirmations INTEGER UNSIGNED NOT NULL,
    difficulty DECIMAL(18,8) NOT NULL,
    hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL ,
    height BIGINT UNSIGNED NOT NULL,
    median_time BIGINT UNSIGNED NOT NULL,
    merkle_root VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    name_claim_root VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    nonce BIGINT UNSIGNED NOT NULL,
    previous_block_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    next_block_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    block_size BIGINT UNSIGNED NOT NULL,
    target VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    block_time BIGINT UNSIGNED NOT NULL,
    version BIGINT UNSIGNED NOT NULL,
    version_hex VARCHAR(10) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    transaction_hashes TEXT NOT NULL,
    transactions_processed TINYINT(1) DEFAULT 0 NOT NULL,


    PRIMARY KEY pk_blockhash (hash),
    CONSTRAINT Cnt_TransactionHashesValidJson CHECK(transaction_hashes IS NULL OR JSON_VALID(transaction_hashes)),
    CONSTRAINT fk_previous_block FOREIGN KEY (previous_block_id) references blocks (hash) ON UPDATE NO ACTION ON DELETE NO ACTION,
    CONSTRAINT fk_next_block FOREIGN KEY (next_block_id) references blocks (hash) ON UPDATE NO ACTION ON DELETE NO ACTION,
    INDEX idx_block_height (height),
    INDEX idx_block_time (block_time),
    INDEX idx_median_time (median_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS transactions
(
    block_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci COMMENT 'named id instead of hash for SQLBoiler' NOT NULL,
    input_count INTEGER UNSIGNED NOT NULL,
    output_count INTEGER UNSIGNED NOT NULL,
    value FLOAT DEFAULT '0.00000000' NOT NULL,
    fee FLOAT DEFAULT '0.00000000' NOT NULL,
    transaction_time BIGINT UNSIGNED,
    transaction_size BIGINT UNSIGNED NOT NULL,
    hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    version INTEGER NOT NULL,
    lock_time INTEGER UNSIGNED NOT NULL,
    raw TEXT,

    created_time INTEGER UNSIGNED NOT NULL,
    PRIMARY KEY pk_transaction (hash),
    CONSTRAINT fk_transaction_block FOREIGN KEY (block_id) REFERENCES blocks (hash),
    INDEX idx_transaction_time (transaction_time),
    INDEX idx_transaction_created_time (created_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS addresses
(
    id SERIAL,

    address VARCHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    first_seen DATETIME,
    total_received DECIMAL(18,8) DEFAULT 0 NOT NULL,
    total_sent DECIMAL(18,8) DEFAULT 0 NOT NULL,
    balance DECIMAL(18,8) AS (total_received - total_sent),
    tag VARCHAR(30) NOT NULL,
    tag_url VARCHAR(200),

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,

    PRIMARY KEY pk_address (id),
    UNIQUE KEY idx_address_address (address),
    UNIQUE KEY idx_address_tag (tag),
    INDEX idx_address_total_received (total_received),
    INDEX idx_address_total_sent (total_sent),
    INDEX idx_address_balance (balance),
    INDEX Idx_address_created (created),
    INDEX Idx_address_modified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS inputs
(
    transaction_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    address_id BIGINT UNSIGNED,
    is_coinbase TINYINT(1) DEFAULT 0 NOT NULL,
    coinbase VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    prevout_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    prevout_n INTEGER UNSIGNED,
    prevout_spend_updated TINYINT(1) DEFAULT 0 NOT NULL,
    sequence_id INTEGER UNSIGNED,
    value DECIMAL(18,8),
    script_sig_ssm TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_sig_hex TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,

    PRIMARY KEY pk_input (transaction_id,sequence_id),
    CONSTRAINT fk_input_transaction FOREIGN KEY  (transaction_id) REFERENCES transactions (hash),
    CONSTRAINT fk_input_address FOREIGN KEY (address_id) REFERENCES addresses (Id),
    INDEX idx_input_value (value),
    INDEX idx_prevout_hash (prevout_hash),
    INDEX idx_input_created (created),
    INDEX idx_input_modified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS outputs
(
    id SERIAL,

    value DECIMAL(18,8),
    v_out INTEGER UNSIGNED,
    transaction_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    type VARCHAR(20) CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_pub_key_asm TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_pub_key_hex TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    required_signatures INTEGER UNSIGNED,
    hash160 VARCHAR(50) CHARACTER SET latin1 COLLATE latin1_general_ci,
    addresslist TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    is_spent TINYINT(1) DEFAULT 0 NOT NULL,
    spent_by_transaction_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    spent_by_transaction_sequence_id INTEGER UNSIGNED,
    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,
    PRIMARY KEY pk_output (id),
    CONSTRAINT fk_output_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (hash),
    CONSTRAINT fk_output_spent_by_input FOREIGN KEY (spent_by_transaction_id,spent_by_transaction_sequence_id) REFERENCES inputs (transaction_id,sequence_id),
    CONSTRAINT cnt_addresslist_valid_json CHECK(addresslist IS NULL OR JSON_VALID(addresses)),
    INDEX idx_output_value (value),
    INDEX idx_ouptut_created (created),
    INDEX idx_output_modified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS outputs_addresses
(
    output_id BIGINT UNSIGNED NOT NULL,
    address_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY pk_output_address (output_id, address_id),
    CONSTRAINT idx_outputs_addresses_output FOREIGN KEY (output_id) REFERENCES outputs (id),
    CONSTRAINT idx_outputs_addresses_address FOREIGN KEY (address_id) REFERENCES addresses (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS transaction_addresses
(
    transaction_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    address_id BIGINT UNSIGNED NOT NULL,
    debit_amount DECIMAL(18,8) DEFAULT 0 NOT NULL COMMENT 'Sum of the inputs to this address for the tx',
    credit_amount DECIMAL(18,8) DEFAULT 0 NOT NULL COMMENT 'Sum of the outputs to this address for the tx',
    latest_transaction_time DATETIME NOT NULL,
    PRIMARY KEY pk_transaction_address (transaction_id, address_id),
    CONSTRAINT idx_transactions_addresses_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (hash),
    CONSTRAINT idx_transactions_addresses_address FOREIGN KEY (address_id) REFERENCES addresses (id),
    INDEX idx_transactions_addresses_latest_transaction_time (latest_transaction_time),
    INDEX idx_transactions_addresses_debit (debit_amount),
    INDEX idx_transactions_addresses_credit (credit_amount)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claims
(
    id SERIAL,
    transaction_of_claim_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci COMMENT 'hash reference of transaction',
    v_out INTEGER UNSIGNED NOT NULL,
    name VARCHAR(1024) NOT NULL,
    claim_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    claim_type TINYINT(1) DEFAULT 0 NOT NULL,  -- 1 - CertificateType, 2 - StreamType
    publisher_claim_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci COMMENT 'references a ClaimId with CertificateType',
    publisher_sig VARCHAR(200) CHARACTER SET latin1 COLLATE latin1_general_ci,
    certificate TEXT,
    transaction_time INTEGER UNSIGNED,
    version VARCHAR(10) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,

    -- Additional fields for easy indexing of stream types
    author VARCHAR(512),
    description MEDIUMTEXT,
    content_type VARCHAR(162) CHARACTER SET latin1 COLLATE latin1_general_ci,
    is_n_s_f_w TINYINT(1) DEFAULT 0 NOT NULL,
    language VARCHAR(20) CHARACTER SET latin1 COLLATE latin1_general_ci,
    thumbnail_url TEXT,
    title TEXT,
    fee DECIMAL(18,8) DEFAULT 0 NOT NULL,
    fee_currency CHAR(3),
    is_filtered TINYINT(1) DEFAULT 0 NOT NULL,

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,
    PRIMARY KEY pk_claim (id),
    CONSTRAINT fk_claim_transaction FOREIGN KEY (transaction_of_claim_id) REFERENCES transactions (hash),
    CONSTRAINT fk_claim_publisher FOREIGN KEY (publisher_claim_id) REFERENCES claims (claim_id),
    UNIQUE KEY idx_claim_unique (transaction_of_claim_id, v_out, claim_id),
    CONSTRAINT Cnt_claimCertificate CHECK(certificate IS NULL OR JSON_VALID(certificate)), -- certificate type
    INDEX Idx_claim (claim_id),
    INDEX Idx_claim_transaction_time (transaction_time),
    INDEX Idx_claim_created (created),
    INDEX Idx_claim_modified (modified),

    INDEX Idx_ClaimAuthor (author(191)),
    INDEX Idx_ClaimContentType (content_type),
    INDEX Idx_ClaimLanguage (language),
    INDEX Idx_ClaimTitle (title(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claim_streams
(
    claim_id BIGINT UNSIGNED NOT NULL,
    stream MEDIUMTEXT NOT NULL,
    PRIMARY KEY pk_claim_stream (claim_id),
    FOREIGN KEY pk_claim_stream_claim (claim_id) REFERENCES claims (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS price_history
(
    id SERIAL,
    b_t_c DECIMAL(18,8) DEFAULT 0 NOT NULL,
    u_s_d DECIMAL(18,2) DEFAULT 0 NOT NULL,
    created DATETIME NOT NULL,
    PRIMARY KEY pk_price_history (id),
    UNIQUE KEY Idx_price_history_created (created)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd