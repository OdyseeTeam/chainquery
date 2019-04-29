-- +migrate Up

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS tag
(
  id SERIAL,
  tag NVARCHAR(255) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY PK_id (id),
  UNIQUE Idx_tag(tag),
  INDEX Idx_OuptutCreated (created_at),
  INDEX Idx_OutputModified (modified_at)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claim_tag
(
  id SERIAL,
  tag_id BIGINT UNSIGNED,
  claim_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY PK_id (id),
  FOREIGN KEY FK_claim (claim_id) REFERENCES claim (claim_id) ON DELETE CASCADE ON UPDATE NO ACTION,
  FOREIGN KEY FK_tag (tag_id) REFERENCES tag (id) ON DELETE SET NULL ON UPDATE CASCADE,
  UNIQUE Idx_claim_tag(tag_id,claim_id),
  INDEX Idx_OuptutCreated (created_at),
  INDEX Idx_OutputModified (modified_at)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TABLE IF NOT EXISTS claim_in_list
(
  id SERIAL,
  list_claim_id  CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci NOT NULL,
  claim_id CHAR(40) CHARACTER SET latin1 COLLATE latin1_general_ci,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  modified_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  PRIMARY KEY PK_id (id),
  FOREIGN KEY FK_list_claim (list_claim_id) REFERENCES claim (claim_id) ON DELETE CASCADE ON UPDATE NO ACTION,
  UNIQUE Idx_claim_tag(list_claim_id,claim_id),
  INDEX Idx_OuptutCreated (created_at),
  INDEX Idx_OutputModified (modified_at)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  ADD COLUMN type VARCHAR(100),
  ADD COLUMN release_time BIGINT UNSIGNED,
  ADD COLUMN source_hash VARCHAR(255),
  ADD COLUMN source_name VARCHAR(255),
  ADD COLUMN source_size BIGINT UNSIGNED,
  ADD COLUMN source_media_type VARCHAR(255),
  ADD COLUMN source_url VARCHAR(255),
  ADD COLUMN frame_width BIGINT UNSIGNED,
  ADD COLUMN frame_height BIGINT UNSIGNED,
  ADD COLUMN duration BIGINT UNSIGNED,
  ADD COLUMN audio_duration BIGINT UNSIGNED,
  ADD COLUMN os VARCHAR(100),
  ADD COLUMN email VARCHAR(255),
  ADD COLUMN website_url VARCHAR(255),
  ADD COLUMN has_claim_list TINYINT(1),
  ADD COLUMN claim_reference VARCHAR(160),
  ADD COLUMN list_type SMALLINT,
  ADD COLUMN claim_id_list JSON,
  ADD COLUMN country VARCHAR(100),
  ADD COLUMN state VARCHAR(100),
  ADD COLUMN city VARCHAR(100),
  ADD COLUMN code VARCHAR(100),
  ADD COLUMN latitude BIGINT,
  ADD COLUMN longitude BIGINT;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE claim
  ADD INDEX Idx_type (type),
  ADD INDEX Idx_release_time (release_time),
  ADD INDEX Idx_source_hash (source_hash),
  ADD INDEX Idx_source_name (source_name),
  ADD INDEX Idx_source_size (source_size),
  ADD INDEX Idx_source_media_type (source_media_type),
  ADD INDEX Idx_source_url (source_url),
  ADD INDEX Idx_frame_width (frame_width),
  ADD INDEX Idx_frame_height (frame_height),
  ADD INDEX Idx_duration (duration),
  ADD INDEX Idx_audio_duration (audio_duration),
  ADD INDEX Idx_os (os),
  ADD INDEX Idx_email (email),
  ADD INDEX Idx_website_url (website_url),
  ADD INDEX Idx_has_claim_list(has_claim_list),
  ADD INDEX Idx_claim_reference(claim_reference),
  ADD INDEX Idx_list_type(list_type),
  ADD INDEX Idx_country (country),
  ADD INDEX Idx_state (state),
  ADD INDEX Idx_city (city),
  ADD INDEX Idx_latitude (latitude),
  ADD INDEX Idx_longitude (longitude);
-- +migrate StatementEnd