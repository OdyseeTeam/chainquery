-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE output
    ADD INDEX Idx_ModifedSpentTxHash (modified_at ASC, is_spent ASC, transaction_hash ASC);
-- +migrate StatementEnd