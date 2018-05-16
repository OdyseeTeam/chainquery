-- +migrate Up
ALTER TABLE claim CHANGE COLUMN `effective_amount` `effective_amount` BIGINT(20) UNSIGNED NOT NULL DEFAULT 0 ;
ALTER TABLE claim CHANGE COLUMN `fee` `fee` DOUBLE(58,8) NOT NULL DEFAULT '0.00000000' ;

