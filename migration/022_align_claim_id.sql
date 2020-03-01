-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim_in_list
    DROP FOREIGN KEY claim_in_list_ibfk_1;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim_tag
    DROP FOREIGN KEY claim_tag_ibfk_1;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
    CHANGE COLUMN claim_id claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL,
    CHANGE COLUMN claim_reference claim_reference VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' DEFAULT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim_in_list
    CHANGE COLUMN claim_id claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' DEFAULT NULL,
    CHANGE COLUMN list_claim_id list_claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci'  NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim_tag
    CHANGE COLUMN claim_id claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim_tag
    ADD CONSTRAINT claim_tag_ibfk_1
        FOREIGN KEY (claim_id)
            REFERENCES claim (claim_id)
            ON DELETE CASCADE
            ON UPDATE NO ACTION;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim_in_list
    ADD CONSTRAINT claim_in_list_ibfk_1
        FOREIGN KEY (list_claim_id)
            REFERENCES claim (claim_id)
            ON DELETE CASCADE
            ON UPDATE NO ACTION;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE output
    CHANGE COLUMN claim_id claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' DEFAULT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE support
    CHANGE COLUMN supported_claim_id supported_claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE abnormal_claim
    CHANGE COLUMN claim_id claim_id VARCHAR(40) CHARACTER SET 'utf8mb4' COLLATE 'utf8mb4_unicode_ci' NOT NULL;
-- +migrate StatementEnd
