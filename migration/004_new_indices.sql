-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE `output` ADD INDEX `Idx_IsSpent` (`is_spent` ASC);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_FeeAddress` (`fee_address` ASC);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD COLUMN `claim_address` VARCHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL DEFAULT '';
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE `claim` ADD INDEX `Idx_ClaimAddress` (`claim_address`);
-- +migrate StatementEnd
