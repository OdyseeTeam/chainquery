package processing

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func TestCheckHandleReorgKeepsMatchingParent(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	parent := testBlock(2, 2, "parent", BlockProcessingStateComplete, 0)
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(2)).
		WillReturnRows(blockRows(parent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(parent.Hash).
		WillReturnRows(transactionRows())

	height, err := checkHandleReorg(3, parent.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if height != 3 {
		t.Fatalf("expected current height to continue, got %d", height)
	}
}

func TestCheckHandleReorgDeletesForkAndReturnsMatchingHeight(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	staleParent := testBlock(2, 2, "stale-parent", BlockProcessingStateComplete, 0)
	canonicalGrandparent := testBlock(1, 1, "canonical-grandparent", BlockProcessingStateComplete, 0)
	transaction := testTransaction(9, staleParent.Hash, "stale-tx", 1, 1)
	fetcher := newReorgFetchRecorder(t, map[uint64]string{
		2: canonicalGrandparent.Hash,
	})

	restore := replaceReorgBlockFetcher(fetcher.fetch)
	defer restore()

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(2)).
		WillReturnRows(blockRows(staleParent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleParent.Hash).
		WillReturnRows(transactionRows(transaction))
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(staleParent.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(1)).
		WillReturnRows(blockRows(canonicalGrandparent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(canonicalGrandparent.Hash).
		WillReturnRows(transactionRows())

	height, err := checkHandleReorg(3, "canonical-parent")
	if err != nil {
		t.Fatal(err)
	}
	if height != 1 {
		t.Fatalf("expected reorg to resume at height 1, got %d", height)
	}
	fetcher.assertCalls(2)
}

func TestCheckHandleReorgDeletesMultipleForkBlocks(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	staleParent := testBlock(3, 3, "stale-parent", BlockProcessingStateComplete, 0)
	staleGrandparent := testBlock(2, 2, "stale-grandparent", BlockProcessingStateComplete, 0)
	canonicalAncestor := testBlock(1, 1, "canonical-ancestor", BlockProcessingStateComplete, 0)
	parentTx := testTransaction(9, staleParent.Hash, "stale-parent-tx", 1, 1)
	fetcher := newReorgFetchRecorder(t, map[uint64]string{
		3: "canonical-grandparent",
		2: canonicalAncestor.Hash,
	})
	logHook := logrustest.NewGlobal()
	defer logHook.Reset()

	restore := replaceReorgBlockFetcher(fetcher.fetch)
	defer restore()

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(3)).
		WillReturnRows(blockRows(staleParent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleParent.Hash).
		WillReturnRows(transactionRows(parentTx))
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(staleParent.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(2)).
		WillReturnRows(blockRows(staleGrandparent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleGrandparent.Hash).
		WillReturnRows(transactionRows())
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(staleGrandparent.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(1)).
		WillReturnRows(blockRows(canonicalAncestor))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(canonicalAncestor.Hash).
		WillReturnRows(transactionRows())

	height, err := checkHandleReorg(4, "canonical-parent")
	if err != nil {
		t.Fatal(err)
	}
	if height != 1 {
		t.Fatalf("expected reorg to resume at height 1, got %d", height)
	}
	fetcher.assertCalls(3, 2)
	assertLastReorgLog(t, logHook.LastEntry(), 2, uint64(4), uint64(1))
}

func TestCheckHandleReorgReturnsDeleteFailure(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	staleParent := testBlock(2, 2, "stale-parent", BlockProcessingStateComplete, 0)
	transaction := testTransaction(9, staleParent.Hash, "stale-tx", 1, 1)
	fetcher := newReorgFetchRecorder(t, map[uint64]string{})

	restore := replaceReorgBlockFetcher(fetcher.fetch)
	defer restore()

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(2)).
		WillReturnRows(blockRows(staleParent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleParent.Hash).
		WillReturnRows(transactionRows(transaction))
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(staleParent.ID).
		WillReturnError(sql.ErrConnDone)

	height, err := checkHandleReorg(3, "canonical-parent")
	if err == nil {
		t.Fatal("expected delete failure")
	}
	if height != 3 {
		t.Fatalf("expected failure to return original height 3, got %d", height)
	}
	fetcher.assertCalls()
}

func TestCheckHandleReorgReturnsGapAfterDeletingFork(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	staleParent := testBlock(2, 2, "stale-parent", BlockProcessingStateComplete, 0)
	transaction := testTransaction(9, staleParent.Hash, "stale-tx", 1, 1)
	fetcher := newReorgFetchRecorder(t, map[uint64]string{
		2: "missing-ancestor",
	})

	restore := replaceReorgBlockFetcher(fetcher.fetch)
	defer restore()

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(2)).
		WillReturnRows(blockRows(staleParent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleParent.Hash).
		WillReturnRows(transactionRows(transaction))
	testDB.mock.ExpectExec(deleteBlock()).
		WithArgs(staleParent.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(1)).
		WillReturnError(sql.ErrNoRows)

	height, err := checkHandleReorg(3, "canonical-parent")
	if err != nil {
		t.Fatal(err)
	}
	if height != 1 {
		t.Fatalf("expected missing ancestor height 1, got %d", height)
	}
	fetcher.assertCalls(2)
}

func TestCheckHandleReorgErrorsAfterDepthLimit(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	const currentHeight uint64 = 102
	const maxReorgDepth = 100

	fetchResponses := make(map[uint64]string, maxReorgDepth)
	expectedCalls := make([]uint64, 0, maxReorgDepth)
	staleParent := testBlock(101, 101, "stale-101", BlockProcessingStateComplete, 0)
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(101)).
		WillReturnRows(blockRows(staleParent))
	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
		WithArgs(staleParent.Hash).
		WillReturnRows(transactionRows())

	for height := uint64(101); height >= 2; height-- {
		block := testBlock(height, height, fmt.Sprintf("stale-%d", height), BlockProcessingStateComplete, 0)
		if height == 101 {
			block = staleParent
		}
		fetchResponses[height] = "unmatched-chain-parent"
		expectedCalls = append(expectedCalls, height)
		testDB.mock.ExpectExec(deleteBlock()).
			WithArgs(block.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		previousBlock := testBlock(height-1, height-1, fmt.Sprintf("stale-%d", height-1), BlockProcessingStateComplete, 0)
		testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
			WithArgs(height - 1).
			WillReturnRows(blockRows(previousBlock))
		testDB.mock.ExpectQuery(selectFrom(model.TableNames.Transaction)).
			WithArgs(previousBlock.Hash).
			WillReturnRows(transactionRows())
	}

	fetcher := newReorgFetchRecorder(t, fetchResponses)
	restore := replaceReorgBlockFetcher(fetcher.fetch)
	defer restore()

	height, err := checkHandleReorg(currentHeight, "canonical-parent")
	if err == nil {
		t.Fatal("expected depth limit error")
	}
	if height != currentHeight {
		t.Fatalf("expected failure to return original height %d, got %d", currentHeight, height)
	}
	fetcher.assertCalls(expectedCalls...)
}

func TestCheckHandleReorgReturnsMissingParentHeight(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Block)).
		WithArgs(uint64(4)).
		WillReturnError(sql.ErrNoRows)

	height, err := checkHandleReorg(5, "chain-parent")
	if err != nil {
		t.Fatal(err)
	}
	if height != 4 {
		t.Fatalf("expected missing parent height 4, got %d", height)
	}
}

func replaceReorgBlockFetcher(fetcher func(*uint64) (*lbrycrd.GetBlockResponse, error)) func() {
	original := fetchBlockForReorg
	fetchBlockForReorg = fetcher
	return func() {
		fetchBlockForReorg = original
	}
}

type reorgFetchRecorder struct {
	t         *testing.T
	responses map[uint64]string
	calls     []uint64
}

func newReorgFetchRecorder(t *testing.T, responses map[uint64]string) *reorgFetchRecorder {
	t.Helper()
	return &reorgFetchRecorder{t: t, responses: responses}
}

func (recorder *reorgFetchRecorder) fetch(height *uint64) (*lbrycrd.GetBlockResponse, error) {
	recorder.t.Helper()
	previousHash, ok := recorder.responses[*height]
	if !ok {
		recorder.t.Fatalf("unexpected reorg block fetch at height %d", *height)
	}
	recorder.calls = append(recorder.calls, *height)
	return &lbrycrd.GetBlockResponse{PreviousBlockHash: previousHash}, nil
}

func (recorder *reorgFetchRecorder) assertCalls(expected ...uint64) {
	recorder.t.Helper()
	if len(recorder.calls) != len(expected) {
		recorder.t.Fatalf("expected fetch calls %v, got %v", expected, recorder.calls)
	}
	for i := range expected {
		if recorder.calls[i] != expected[i] {
			recorder.t.Fatalf("expected fetch calls %v, got %v", expected, recorder.calls)
		}
	}
}

func assertLastReorgLog(t *testing.T, entry *logrus.Entry, depth int, height uint64, lastMatchingHeight uint64) {
	t.Helper()
	if entry == nil {
		t.Fatal("expected reorg log entry")
	}
	if entry.Data["depth"] != depth {
		t.Fatalf("expected reorg depth %d, got %v", depth, entry.Data["depth"])
	}
	if entry.Data["height"] != height {
		t.Fatalf("expected reorg height %d, got %v", height, entry.Data["height"])
	}
	if entry.Data["last_matching_height"] != lastMatchingHeight {
		t.Fatalf("expected last matching height %d, got %v", lastMatchingHeight, entry.Data["last_matching_height"])
	}
}
