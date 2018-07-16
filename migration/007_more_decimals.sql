-- +migrate Up
ALTER TABLE block CHANGE COLUMN difficulty difficulty DOUBLE(50,8) NOT NULL ;