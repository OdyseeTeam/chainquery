-- +migrate Up

-- +migrate StatementBegin
CREATE TABLE purchase
(
    id SERIAL,
    transaction_by_hash_id VARCHAR(70) CHARACTER SET latin1 COLLATE latin1_general_ci,
    vout INTEGER UNSIGNED NOT NULL,
    claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci',
    publisher_id CHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci',
    height INTEGER UNSIGNED NOT NULL,
    amount_satoshi BIGINT DEFAULT 0 NOT NULL,

    created DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY PK_Purchase (id),
    FOREIGN KEY FK_PurchaseTransaction (transaction_by_hash_id) REFERENCES transaction (hash) ON DELETE CASCADE ON UPDATE NO ACTION,
    UNIQUE KEY Idx_PurchaseUnique (transaction_by_hash_id, vout, claim_id),
    INDEX Idx_PurchaseClaim (claim_id),
    INDEX Idx_PurchaseAmount (amount_satoshi),
    INDEX Idx_PurchaseCreated (created),
    INDEX Idx_PurchaseModified (modified)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;
-- +migrate StatementEnd