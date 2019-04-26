-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE input ADD COLUMN vin INTEGER UNSIGNED;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE input CHANGE COLUMN coinbase coinbase VARCHAR(255) CHARACTER SET 'latin1' COLLATE 'latin1_general_ci' NULL DEFAULT NULL;
-- +migrate StatementEnd