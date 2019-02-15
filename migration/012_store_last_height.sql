-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE job_status ADD COLUMN state JSON;
-- +migrate StatementEnd