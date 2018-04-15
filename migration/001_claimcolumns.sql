-- +migrate Up
CREATE TABLE IF NOT EXISTS `application_status`
(
  `id` SERIAL,
  `app_version` INT NOT NULL,
  `data_version`INT NOT NULL,
  `api_version`INT NOT NULL,

  PRIMARY KEY `PK_id` (`id`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_unicode_ci ROW_FORMAT=COMPRESSED KEY_BLOCK_SIZE=4;

ALTER TABLE claim MODIFY COLUMN `fee_currency` CHAR(30);
ALTER TABLE claim MODIFY COLUMN `language` CHAR(30);
ALTER TABLE claim ADD COLUMN `valid_at_height` INTEGER UNSIGNED NOT NULL;
ALTER TABLE claim ADD COLUMN `effective_amount` DOUBLE(18,8) DEFAULT 0 NOT NULL;