-- +migrate Up

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS blocks (
    id SERIAL,

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
    previous_block_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    next_block_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    block_size BIGINT UNSIGNED NOT NULL,
    target VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    block_time BIGINT UNSIGNED NOT NULL,
    version BIGINT UNSIGNED NOT NULL,
    version_hex VARCHAR(10) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    transaction_hashes TEXT,
    transactions_processed TINYINT(1) DEFAULT 0 NOT NULL,

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,

    PRIMARY KEY PK_Block (id),
    UNIQUE KEY Idx_BlockHash (hash),
    CONSTRAINT Cnt_TransactionHashesValidJson CHECK(transaction_hashes IS NULL OR JSON_VALID(transaction_hashes)),
    INDEX Idx_BlockHeight (height),
    INDEX Idx_BlockTime (block_time),
    INDEX Idx_MedianTime (median_time),
    INDEX Idx_PreviousBlockHash (previous_block_hash),
    INDEX Idx_BlockCreated (created),
    INDEX Idx_BlockModified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS transactions
(
    id SERIAL,

    block_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    input_count INTEGER UNSIGNED NOT NULL,
    output_count INTEGER UNSIGNED NOT NULL,
    value DECIMAL(18,8) NOT NULL,
    fee DECIMAL(18,8) DEFAULT 0 NOT NULL,
    transaction_time BIGINT UNSIGNED,
    transaction_size BIGINT UNSIGNED NOT NULL,
    hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    version INTEGER NOT NULL,
    lock_time INTEGER UNSIGNED NOT NULL,
    raw TEXT,

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,

    created_time INTEGER UNSIGNED NOT NULL,
    PRIMARY KEY PK_Transaction (id),
    FOREIGN KEY FK_TransactionBlockHash (block_hash) REFERENCES blocks (hash),
    UNIQUE KEY Idx_TransactionHash (hash),
    INDEX Idx_TransactionTime (transaction_time),
    INDEX Idx_TransactionCreatedTime (created_time),
    INDEX Idx_TransactionCreated (created),
    INDEX Idx_TransactionModified (modified)
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

    PRIMARY KEY PK_Address (id),
    UNIQUE KEY Idx_AddressAddress (address),
    UNIQUE KEY Idx_AddressTag (tag),
    INDEX Idx_AddressTotalReceived (total_received),
    INDEX Idx_AddressTotalSent (total_sent),
    INDEX Idx_AddressBalance (balance),
    INDEX Idx_AddressCreated (created),
    INDEX Idx_AddressModified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS inputs
(
    id SERIAL,

    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    address_id BIGINT UNSIGNED,
    is_coinbase TINYINT(1) DEFAULT 0 NOT NULL,
    coinbase VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    prevout_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    prevout_n INTEGER UNSIGNED,
    prevout_spend_updated TINYINT(1) DEFAULT 0 NOT NULL,
    sequence INTEGER UNSIGNED,
    value DECIMAL(18,8),
    script_sig_ssm TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_sig_hex TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,

    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,

    PRIMARY KEY PK_Input (id),
    FOREIGN KEY FK_InputAddress (address_id) REFERENCES addresses (id),
    FOREIGN KEY FK_InputTransaction (transaction_id) REFERENCES transactions (id),
    INDEX Idx_InputValue (value),
    INDEX Idx_PrevoutHash (prevout_hash),
    INDEX Idx_InputCreated (created),
    INDEX Idx_InputModified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS input_addresses
(
    input_id BIGINT UNSIGNED NOT NULL,
    address_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY PK_InputAddress (input_id, address_id),
    FOREIGN KEY Idx_InputsAddressesInput (input_id) REFERENCES inputs (id),
    FOREIGN KEY Idx_InputsAddressesAddress (address_id) REFERENCES addresses (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS outputs
(
    id SERIAL,
    transaction_id BIGINT UNSIGNED NOT NULL,
    value DECIMAL(18,8),
    v_out INTEGER UNSIGNED,
    type VARCHAR(20) CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_pub_key_asm TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    script_pub_key_hex TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    required_signatures INTEGER UNSIGNED,
    hash160 VARCHAR(50) CHARACTER SET latin1 COLLATE latin1_general_ci,
    addresses TEXT CHARACTER SET latin1 COLLATE latin1_general_ci,
    is_spent TINYINT(1) DEFAULT 0 NOT NULL,
    spent_by_input_id BIGINT UNSIGNED,
    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,
    PRIMARY KEY PK_Output (id),
    FOREIGN KEY FK_OutputTransaction (transaction_id) REFERENCES transactions (id),
    FOREIGN KEY FK_OutputSpentByInput (spent_by_input_id) REFERENCES inputs (id),
    CONSTRAINT Cnt_AddressesValidJson CHECK(addresses IS NULL OR JSON_VALID(addresses)),
    INDEX Idx_OutputValue (value),
    INDEX Idx_OuptutCreated (created),
    INDEX Idx_OutputModified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS outputs_addresses
(
    output_id BIGINT UNSIGNED NOT NULL,
    address_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY PK_OutputAddress (output_id, address_id),
    FOREIGN KEY Idx_OutputsAddressesOutput (output_id) REFERENCES outputs (id),
    FOREIGN KEY Idx_OutputsAddressesAddress (address_id) REFERENCES addresses (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS transactions_addresses
(
    transaction_id BIGINT UNSIGNED NOT NULL,
    address_id BIGINT UNSIGNED NOT NULL,
    debit_amount DECIMAL(18,8) DEFAULT 0 NOT NULL COMMENT 'Sum of the inputs to this address for the tx',
    credit_amount DECIMAL(18,8) DEFAULT 0 NOT NULL COMMENT 'Sum of the outputs to this address for the tx',
    latest_transaction_time DATETIME NOT NULL,
    PRIMARY KEY PK_TransactionAddress (transaction_id, address_id),
    FOREIGN KEY Idx_TransactionsAddressesTransaction (transaction_id) REFERENCES transactions (id),
    FOREIGN KEY Idx_TransactionsAddressesAddress (address_id) REFERENCES addresses (id),
    INDEX Idx_TransactionsAddressesLatestTransactionTime (latest_transaction_time),
    INDEX Idx_TransactionsAddressesDebit (debit_amount),
    INDEX Idx_TransactionsAddressesCredit (credit_amount)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claims
(
    id SERIAL,
    transaction_hash VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    v_out INTEGER UNSIGNED NOT NULL,
    name VARCHAR(1024) NOT NULL,
    claim_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
    claim_type TINYINT(1) DEFAULT 0 NOT NULL,  -- 1 - CertificateType, 2 - StreamType
    publisher_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci COMMENT 'references a ClaimId with CertificateType',
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
    PRIMARY KEY PK_Claim (id),
    FOREIGN KEY FK_ClaimTransaction (transaction_hash) REFERENCES transactions (hash),
    FOREIGN KEY FK_ClaimPublisher (publisher_id) REFERENCES claims (claim_id),
    UNIQUE KEY Idx_ClaimUnique (transaction_hash, v_out, claim_id),
    CONSTRAINT Cnt_ClaimCertificate CHECK(certificate IS NULL OR JSON_VALID(certificate)), -- certificate type
    INDEX Idx_Claim (claim_id),
    INDEX Idx_ClaimTransactionTime (transaction_time),
    INDEX Idx_ClaimCreated (created),
    INDEX Idx_ClaimModified (modified),

    INDEX Idx_ClaimAuthor (author(191)),
    INDEX Idx_ClaimContentType (content_type),
    INDEX Idx_ClaimLanguage (language),
    INDEX Idx_ClaimTitle (title(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claim_streams
(
    id BIGINT UNSIGNED NOT NULL,
    stream MEDIUMTEXT NOT NULL,
    PRIMARY KEY PK_ClaimStream (id),
    FOREIGN KEY PK_ClaimStreamClaim (id) REFERENCES claims (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS price_history
(
    id SERIAL,
    b_t_c DECIMAL(18,8) DEFAULT 0 NOT NULL,
    u_s_d DECIMAL(18,2) DEFAULT 0 NOT NULL,
    created DATETIME NOT NULL,
    PRIMARY KEY PK_PriceHistory (id),
    UNIQUE KEY Idx_PriceHistoryCreated (created)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd