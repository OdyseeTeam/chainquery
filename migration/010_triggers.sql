-- +migrate Up
-- +migrate StatementBegin
ALTER TABLE address ADD COLUMN balance DOUBLE(58,8) NOT NULL DEFAULT '0.00000000' ;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction ADD COLUMN value DOUBLE(58,8) NOT NULL DEFAULT '0.00000000' ;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_insert_value AFTER INSERT ON transaction_address
  FOR EACH ROW
  UPDATE transaction
  SET transaction.value = transaction.value + NEW.credit_amount
  WHERE transaction.id = NEW.transaction_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_update_value AFTER UPDATE ON transaction_address
  FOR EACH ROW
  UPDATE transaction
  SET transaction.value = transaction.value - OLD.credit_amount + NEW.credit_amount
  WHERE transaction.id = NEW.transaction_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_insert_balance AFTER INSERT ON transaction_address
  FOR EACH ROW
  UPDATE address
  SET address.balance = address.balance + (NEW.credit_amount - NEW.debit_amount)
  WHERE address.id = NEW.address_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_update_balance AFTER UPDATE ON transaction_address
  FOR EACH ROW
  UPDATE address
  SET address.balance = address.balance - (OLD.credit_amount - OLD.debit_amount) + (NEW.credit_amount - NEW.debit_amount)
  WHERE address.id = NEW.address_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_delete_balance AFTER DELETE ON transaction_address
  FOR EACH ROW
  UPDATE address
  SET address.balance = address.balance - (OLD.credit_amount - OLD.debit_amount)
  WHERE address.id = OLD.address_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE address
SET address.balance = (SELECT COALESCE( SUM( ta.credit_amount - ta.debit_amount ),0.0) FROM transaction_address ta WHERE ta.address_id = address.id);
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE transaction
SET transaction.value = ( SELECT COALESCE( SUM( ta.credit_amount ),0.0) FROM transaction_address ta WHERE ta.transaction_id = transaction.id);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE address ADD INDEX (balance);
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction ADD INDEX (value);
-- +migrate StatementEnd