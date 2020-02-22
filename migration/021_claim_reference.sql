-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim CHANGE COLUMN claim_reference claim_reference CHAR(40) CHARACTER SET 'latin1' COLLATE 'latin1_general_ci' NULL DEFAULT NULL ;
-- +migrate StatementEnd