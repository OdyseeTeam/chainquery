-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `output` ADD INDEX `Idx_IsSpent` (`is_spent` ASC);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_FeeAddress` (`fee_address` ASC);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `output` ADD INDEX `Idx_SpentOutput` (`transaction_hash` ASC, `vout` ASC, `is_spent` ASC) COMMENT 'used for grabbing spent outputs with joins';
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_ClaimOutpoint` (`transaction_by_hash_id` ASC, `vout` ASC) COMMENT 'used for match claim to output with joins';
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD COLUMN `claim_address` VARCHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL DEFAULT '';
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_ClaimAddress` (`claim_address`);
-- +migrate StatementEnd
