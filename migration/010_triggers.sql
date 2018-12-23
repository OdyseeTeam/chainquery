-- +migrate Up
-- +migrate StatementBegin
ALTER TABLE address ADD COLUMN balance DOUBLE(58,8) NOT NULL DEFAULT '0.00000000' ;
-- +migrate StatementEnd

-- +migrate StatementBegin
ALTER TABLE transaction ADD COLUMN value DOUBLE(58,8) NOT NULL DEFAULT '0.00000000' ;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_update_balance AFTER INSERT ON transaction_address
  FOR EACH ROW
  UPDATE address
  SET address.balance = ( SELECT SUM( ta.credit_amount - ta.debit_amount ) FROM transaction_address ta WHERE ta.address_id = NEW.address_id)
  WHERE address.id = NEW.address_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_update_value AFTER INSERT ON transaction_address
  FOR EACH ROW
  UPDATE transaction
  SET transaction.value = ( SELECT SUM( ta.credit_amount - ta.debit_amount ) FROM transaction_address ta WHERE ta.transaction_id = NEW.transaction_id)
  WHERE transaction.id = NEW.transaction_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE TRIGGER tg_delete_balance AFTER DELETE ON transaction_address
  FOR EACH ROW
  UPDATE address
  SET address.balance = ( SELECT SUM( ta.credit_amount - ta.debit_amount ) FROM transaction_address ta WHERE ta.address_id = NEW.address_id)
  WHERE address.id = NEW.address_id;
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE address
SET address.balance = ( SELECT SUM( ta.credit_amount - ta.debit_amount ) FROM transaction_address ta WHERE ta.address_id = address.id)
-- +migrate StatementEnd

-- +migrate StatementBegin
UPDATE transaction
SET transaction.value = ( SELECT SUM( ta.credit_amount - ta.debit_amount ) FROM transaction_address ta WHERE ta.transaction_id = transaction.id)
-- +migrate StatementEnd