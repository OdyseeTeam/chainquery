-- +migrate Up

-- +migrate StatementBegin
ALTER TABLE claim
  ADD COLUMN license NVARCHAR(255),
  ADD COLUMN license_url NVARCHAR(255),
  ADD COLUMN preview NVARCHAR(255);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
    ADD INDEX Idx_License (license),
    ADD INDEX Idx_LicenseURL (license_url),
    ADD INDEX Idx_Preview (preview);
-- +migrate StatementEnd