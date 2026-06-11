package processing

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lbryio/chainquery/model"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type sqlBoilerTestDB struct {
	db   *sql.DB
	mock sqlmock.Sqlmock
	old  boil.Executor
}

func newSQLBoilerTestDB(t *testing.T) *sqlBoilerTestDB {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	testDB := &sqlBoilerTestDB{
		db:   db,
		mock: mock,
		old:  boil.GetDB(),
	}
	boil.SetDB(db)
	return testDB
}

func (testDB *sqlBoilerTestDB) close(t *testing.T) {
	t.Helper()
	testDB.mock.ExpectClose()
	err := testDB.db.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = testDB.mock.ExpectationsWereMet()
	if err != nil {
		t.Fatal(err)
	}
	boil.SetDB(testDB.old)
}

func TestCleanupIncompleteHeadDeletesIncompleteHead(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	head := testBlock(2, 2, "head", BlockProcessingStateProcessing, 0)
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(MempoolBlockHash).
		WillReturnRows(blockRows(head))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(transactionRows())
	testDB.mock.ExpectExec(updateBlockProcessingState()).
		WithArgs(BlockProcessingStateIncomplete, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := cleanupIncompleteHead()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanupIncompleteHeadOnlyHandlesCurrentHead(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	head := testBlock(2, 2, "head", BlockProcessingStateComplete, 0)
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(MempoolBlockHash).
		WillReturnRows(blockRows(head))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(transactionRows())

	err := CleanupIncompleteHead()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanupIncompleteHeadTreatsGenesisAsNormalHead(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	genesis := testBlock(1, 0, "genesis", "", 0)
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(MempoolBlockHash).
		WillReturnRows(blockRows(genesis))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(transactionRows())
	testDB.mock.ExpectExec(updateBlockProcessingState()).
		WithArgs(BlockProcessingStateComplete, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := cleanupIncompleteHead()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBackfillLegacyBlockStatesBatchesAndExcludesHeadAndMempool(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	head := testBlock(3, 3, "head", BlockProcessingStateComplete, 0)
	legacy := testBlock(2, 2, "legacy", "", 1)
	transaction := testTransaction(9, legacy.Hash, "tx", 1, 1)

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(MempoolBlockHash).
		WillReturnRows(blockRows(head))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(MempoolBlockHash, sqlmock.AnyArg()).
		WillReturnRows(blockRows(legacy))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(transactionRows(transaction))
	testDB.mock.ExpectQuery(countFrom(model.TableNames.Input)).
		WithArgs(transaction.Hash).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	testDB.mock.ExpectQuery(countFrom(model.TableNames.Output)).
		WithArgs(transaction.Hash).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	testDB.mock.ExpectExec(updateBlockProcessingState()).
		WithArgs(BlockProcessingStateComplete, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	count, err := backfillLegacyBlockStates(1)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one backfilled block, got %d", count)
	}
}

func TestDeleteBlockWithRetryReturnsDeleteFailure(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	oldDelay := blockDeleteRetryDelay
	blockDeleteRetryDelay = 0
	defer restoreBlockDeleteRetryDelay(oldDelay)

	block := testBlock(5, 5, "bad", BlockProcessingStateIncomplete, 0)
	for attempt := 0; attempt < blockDeleteRetryAttempts; attempt++ {
		testDB.mock.ExpectExec(deleteBlock()).
			WithArgs(sqlmock.AnyArg()).
			WillReturnError(sql.ErrConnDone)
	}

	err := deleteBlockWithRetry(block)
	if err == nil {
		t.Fatal("expected delete failure")
	}
}

func restoreBlockDeleteRetryDelay(delay time.Duration) {
	blockDeleteRetryDelay = delay
}

func selectFrom(table string) string {
	return "SELECT .* FROM " + regexp.QuoteMeta("`"+table+"`")
}

func countFrom(table string) string {
	return "SELECT COUNT\\(\\*\\) FROM " + regexp.QuoteMeta("`"+table+"`")
}

func updateBlockProcessingState() string {
	query := "UPDATE `" + model.TableNames.Block + "` SET `" + model.BlockColumns.ProcessingState + "`=? WHERE `" + model.BlockColumns.ID + "`=?"
	return regexp.QuoteMeta(query)
}

func deleteBlock() string {
	query := "DELETE FROM `" + model.TableNames.Block + "` WHERE `" + model.BlockColumns.ID + "`=?"
	return regexp.QuoteMeta(query)
}

func testBlock(id uint64, height uint64, hash string, state string, txCount int) *model.Block {
	block := &model.Block{
		ID:              id,
		Bits:            "bits",
		Chainwork:       "chainwork",
		Confirmations:   1,
		Difficulty:      1,
		Hash:            hash,
		Height:          height,
		MerkleRoot:      "merkle",
		NameClaimRoot:   "claims",
		Nonce:           1,
		BlockSize:       1,
		BlockTime:       1,
		Version:         1,
		VersionHex:      "1",
		TXCount:         txCount,
		CreatedAt:       time.Unix(1, 0),
		ModifiedAt:      time.Unix(1, 0),
		ProcessingState: null.NewString(state, state != ""),
	}
	if height > 0 {
		block.PreviousBlockHash.SetValid("previous")
	}
	return block
}

func blockRows(blocks ...*model.Block) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		model.BlockColumns.ID,
		model.BlockColumns.Bits,
		model.BlockColumns.Chainwork,
		model.BlockColumns.Confirmations,
		model.BlockColumns.Difficulty,
		model.BlockColumns.Hash,
		model.BlockColumns.Height,
		model.BlockColumns.MerkleRoot,
		model.BlockColumns.NameClaimRoot,
		model.BlockColumns.Nonce,
		model.BlockColumns.PreviousBlockHash,
		model.BlockColumns.NextBlockHash,
		model.BlockColumns.BlockSize,
		model.BlockColumns.BlockTime,
		model.BlockColumns.Version,
		model.BlockColumns.VersionHex,
		model.BlockColumns.TXCount,
		model.BlockColumns.ProcessingState,
		model.BlockColumns.CreatedAt,
		model.BlockColumns.ModifiedAt,
	})
	for _, block := range blocks {
		rows.AddRow(
			int64(block.ID),
			block.Bits,
			block.Chainwork,
			int64(block.Confirmations),
			block.Difficulty,
			block.Hash,
			int64(block.Height),
			block.MerkleRoot,
			block.NameClaimRoot,
			int64(block.Nonce),
			nullStringValue(block.PreviousBlockHash),
			nullStringValue(block.NextBlockHash),
			int64(block.BlockSize),
			int64(block.BlockTime),
			int64(block.Version),
			block.VersionHex,
			int64(block.TXCount),
			nullStringValue(block.ProcessingState),
			block.CreatedAt,
			block.ModifiedAt,
		)
	}
	return rows
}

func testTransaction(id uint64, blockHash string, hash string, inputCount uint, outputCount uint) *model.Transaction {
	return &model.Transaction{
		ID:              id,
		BlockHashID:     null.StringFrom(blockHash),
		InputCount:      inputCount,
		OutputCount:     outputCount,
		TransactionTime: null.Uint64From(1),
		TransactionSize: 1,
		Hash:            hash,
		Version:         1,
		LockTime:        0,
		CreatedAt:       time.Unix(1, 0),
		ModifiedAt:      time.Unix(1, 0),
		CreatedTime:     time.Unix(1, 0),
		Value:           1,
	}
}

func transactionRows(transactions ...*model.Transaction) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		model.TransactionColumns.ID,
		model.TransactionColumns.BlockHashID,
		model.TransactionColumns.InputCount,
		model.TransactionColumns.OutputCount,
		model.TransactionColumns.TransactionTime,
		model.TransactionColumns.TransactionSize,
		model.TransactionColumns.Hash,
		model.TransactionColumns.Version,
		model.TransactionColumns.LockTime,
		model.TransactionColumns.CreatedAt,
		model.TransactionColumns.ModifiedAt,
		model.TransactionColumns.CreatedTime,
		model.TransactionColumns.Value,
	})
	for _, transaction := range transactions {
		rows.AddRow(
			int64(transaction.ID),
			nullStringValue(transaction.BlockHashID),
			int64(transaction.InputCount),
			int64(transaction.OutputCount),
			nullUint64Value(transaction.TransactionTime),
			int64(transaction.TransactionSize),
			transaction.Hash,
			int64(transaction.Version),
			int64(transaction.LockTime),
			transaction.CreatedAt,
			transaction.ModifiedAt,
			transaction.CreatedTime,
			transaction.Value,
		)
	}
	return rows
}

func nullStringValue(value null.String) interface{} {
	if !value.Valid {
		return nil
	}
	return value.String
}

func nullUint64Value(value null.Uint64) interface{} {
	if !value.Valid {
		return nil
	}
	return int64(value.Uint64)
}
