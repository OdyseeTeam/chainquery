-- +migrate Up

-- +migrate StatementBegin
CREATE INDEX idx_support_modified_at ON support (modified_at);
-- +migrate StatementEnd