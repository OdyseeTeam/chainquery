-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim CHANGE content_type content_type VARCHAR(162) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NULL DEFAULT NULL;
-- +migrate StatementEnd