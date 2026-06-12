package processing

import (
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/proto"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	lbryerrors "github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/extras/stop"
	pb "github.com/lbryio/types/v2/go"
	"github.com/volatiletech/null/v8"
)

func TestBlockTxGraphDeduplicatesSameParentInputs(t *testing.T) {
	parent := &lbrycrd.TxRawResult{Txid: "parent"}
	child := &lbrycrd.TxRawResult{Txid: "child", Vin: []lbrycrd.Vin{
		{TxID: "parent", Vout: 0},
		{TxID: "parent", Vout: 1},
	}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{parent, child}, map[string]*lbrycrd.TxRawResult{
		"parent": parent,
		"child":  child,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved["child"] != 1 {
		t.Fatalf("expected one unresolved parent, got %d", graph.unresolved["child"])
	}
}

func TestBlockTxGraphLinearAndFanInDependencies(t *testing.T) {
	parent := &lbrycrd.TxRawResult{Txid: "parent"}
	middle := &lbrycrd.TxRawResult{Txid: "middle", Vin: []lbrycrd.Vin{{TxID: "parent"}}}
	child := &lbrycrd.TxRawResult{Txid: "child", Vin: []lbrycrd.Vin{
		{TxID: "parent"},
		{TxID: "middle"},
	}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{parent, middle, child}, map[string]*lbrycrd.TxRawResult{
		"parent": parent,
		"middle": middle,
		"child":  child,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[middle.Txid] != 1 {
		t.Fatalf("expected middle to have one dependency, got %d", graph.unresolved[middle.Txid])
	}
	if graph.unresolved[child.Txid] != 2 {
		t.Fatalf("expected child to have two dependencies, got %d", graph.unresolved[child.Txid])
	}
}

func TestSchedulerDoesNotDispatchChildBeforeParentCompletes(t *testing.T) {
	disableSchedulerCleanup(t)
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 2
	parentDone := make(chan struct{})
	childStarted := make(chan struct{}, 1)
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		if tx.Txid == "parent" {
			time.Sleep(20 * time.Millisecond)
			close(parentDone)
			return nil
		}
		select {
		case <-parentDone:
		default:
			t.Error("child started before parent completed")
		}
		childStarted <- struct{}{}
		return nil
	}
	defer restoreSchedulerTestGlobals(originalProcessTx, originalMaxParallel)

	parent := &lbrycrd.TxRawResult{Txid: "parent"}
	child := &lbrycrd.TxRawResult{Txid: "child", Vin: []lbrycrd.Vin{{TxID: "parent"}}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{parent, child}, map[string]*lbrycrd.TxRawResult{
		"parent": parent,
		"child":  child,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	err = runBlockTxScheduler(stop.New(nil), graph, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-childStarted:
	default:
		t.Fatal("expected child to be processed")
	}
}

func TestSchedulerKeepsChildBlockedUntilRetrySucceeds(t *testing.T) {
	disableSchedulerCleanup(t)
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	originalMaxFailures := MaxFailures
	originalRetryBackoff := txRetryBackoff
	MaxParallelTxProcessing = 2
	MaxFailures = 1
	txRetryBackoff = time.Millisecond
	attempts := make(map[string]int)
	var mutex sync.Mutex
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		mutex.Lock()
		attempts[tx.Txid]++
		parentAttempts := attempts["parent"]
		mutex.Unlock()
		if tx.Txid == "parent" && parentAttempts == 1 {
			return &mysql.MySQLError{Number: mysqlErrorDeadlock}
		}
		if tx.Txid == "child" && parentAttempts < 2 {
			t.Error("child started before parent retry succeeded")
		}
		return nil
	}
	defer restoreSchedulerRetryTestGlobals(originalProcessTx, originalMaxParallel, originalMaxFailures, originalRetryBackoff)

	parent := &lbrycrd.TxRawResult{Txid: "parent"}
	child := &lbrycrd.TxRawResult{Txid: "child", Vin: []lbrycrd.Vin{{TxID: "parent"}}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{parent, child}, map[string]*lbrycrd.TxRawResult{
		"parent": parent,
		"child":  child,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	err = runBlockTxScheduler(stop.New(nil), graph, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if attempts["parent"] != 2 {
		t.Fatalf("expected parent to retry once, got %d attempts", attempts["parent"])
	}
}

func TestSchedulerDoesNotRetryNonRetryableFailure(t *testing.T) {
	disableSchedulerCleanup(t)
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	originalMaxFailures := MaxFailures
	MaxParallelTxProcessing = 1
	MaxFailures = 10
	attempts := 0
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		attempts++
		return errors.New("deterministic failure")
	}
	defer restoreSchedulerRetryTestGlobals(originalProcessTx, originalMaxParallel, originalMaxFailures, txRetryBackoff)

	tx := &lbrycrd.TxRawResult{Txid: "tx"}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{tx}, map[string]*lbrycrd.TxRawResult{"tx": tx})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	err = runBlockTxScheduler(stop.New(nil), graph, 1, 2)
	if err == nil {
		t.Fatal("expected terminal error")
	}
	if attempts != 1 {
		t.Fatalf("expected one attempt, got %d", attempts)
	}
}

func TestMissingSourceOutputClassificationSurvivesWrapping(t *testing.T) {
	var err error = &MissingSourceOutputError{PrevoutTxID: "prev", PrevoutN: 1}
	err = enrichMissingSourceOutput(err, "current", 7)
	wrapped := lbryerrors.Prefix("Vin Address Creation Error", err)
	missing, ok := missingSourceOutputFromError(wrapped)
	if !ok {
		t.Fatal("expected missing source output classification")
	}
	if missing.PrevoutTxID != "prev" || missing.PrevoutN != 1 || missing.TxID != "current" || missing.BlockHeight != 7 {
		t.Fatalf("unexpected missing source output details: %+v", missing)
	}
}

func TestSyncTransactionsDependencyAwareAcceptsNilStopper(t *testing.T) {
	disableSchedulerCleanup(t)
	originalFetchRawTransaction := fetchRawTransaction
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 1
	processed := make(chan string, 1)
	fetchRawTransaction = func(txID string) (*lbrycrd.TxRawResult, error) {
		return &lbrycrd.TxRawResult{Txid: txID}, nil
	}
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		processed <- tx.Txid
		return nil
	}
	defer restoreSchedulerEndToEndTestGlobals(originalFetchRawTransaction, originalProcessTx, originalMaxParallel)

	err := syncTransactionsDependencyAware(nil, []string{"tx"}, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case txID := <-processed:
		if txID != "tx" {
			t.Fatalf("expected tx to be processed, got %s", txID)
		}
	default:
		t.Fatal("expected transaction to be processed")
	}
}

func TestClaimDependencyCreateThenSupport(t *testing.T) {
	createTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("01"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(claimNameScriptHex()),
		}},
	}
	claimID, err := lbrycrd.ClaimIDFromOutpoint(createTx.Txid, 0)
	if err != nil {
		t.Fatal(err)
	}
	supportTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("02"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(supportScriptHex(claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{createTx, supportTx}, map[string]*lbrycrd.TxRawResult{
		createTx.Txid:  createTx,
		supportTx.Txid: supportTx,
	})
	err = graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[supportTx.Txid] != 1 {
		t.Fatalf("expected support to depend on claim create, got %d dependencies", graph.unresolved[supportTx.Txid])
	}
}

func TestClaimDependencyCreateThenPurchase(t *testing.T) {
	createTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("05"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(claimNameScriptHex()),
		}},
	}
	claimID, err := lbrycrd.ClaimIDFromOutpoint(createTx.Txid, 0)
	if err != nil {
		t.Fatal(err)
	}
	purchaseTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("06"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(purchaseScriptHex(t, claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{createTx, purchaseTx}, map[string]*lbrycrd.TxRawResult{
		createTx.Txid:   createTx,
		purchaseTx.Txid: purchaseTx,
	})
	err = graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[purchaseTx.Txid] != 1 {
		t.Fatalf("expected purchase to depend on claim create, got %d dependencies", graph.unresolved[purchaseTx.Txid])
	}
}

func TestClaimDependencyEarlierSupportBeforeLaterCreate(t *testing.T) {
	createTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("03"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(claimNameScriptHex()),
		}},
	}
	claimID, err := lbrycrd.ClaimIDFromOutpoint(createTx.Txid, 0)
	if err != nil {
		t.Fatal(err)
	}
	supportTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("04"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(supportScriptHex(claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{supportTx, createTx}, map[string]*lbrycrd.TxRawResult{
		supportTx.Txid: supportTx,
		createTx.Txid:  createTx,
	})
	err = graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[createTx.Txid] != 1 {
		t.Fatalf("expected later claim create to depend on earlier support, got %d dependencies", graph.unresolved[createTx.Txid])
	}
}

func TestClaimDependencyRepeatedUpdates(t *testing.T) {
	claimID := strings.Repeat("1", 40)
	firstUpdateTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("07"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(updateScriptHex(claimID)),
		}},
	}
	secondUpdateTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("08"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(updateScriptHex(claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{firstUpdateTx, secondUpdateTx}, map[string]*lbrycrd.TxRawResult{
		firstUpdateTx.Txid:  firstUpdateTx,
		secondUpdateTx.Txid: secondUpdateTx,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[secondUpdateTx.Txid] != 1 {
		t.Fatalf("expected second update to depend on first update, got %d dependencies", graph.unresolved[secondUpdateTx.Txid])
	}
}

func TestClaimDependencyEarlierPurchaseBeforeLaterUpdate(t *testing.T) {
	claimID := strings.Repeat("2", 40)
	purchaseTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("09"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(purchaseScriptHex(t, claimID)),
		}},
	}
	updateTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0a"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(updateScriptHex(claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{purchaseTx, updateTx}, map[string]*lbrycrd.TxRawResult{
		purchaseTx.Txid: purchaseTx,
		updateTx.Txid:   updateTx,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[updateTx.Txid] != 1 {
		t.Fatalf("expected later update to depend on earlier purchase, got %d dependencies", graph.unresolved[updateTx.Txid])
	}
}

func TestClaimDependencyWriterReaderWriter(t *testing.T) {
	claimID := strings.Repeat("3", 40)
	firstUpdateTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0b"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(updateScriptHex(claimID)),
		}},
	}
	purchaseTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0c"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(purchaseScriptHex(t, claimID)),
		}},
	}
	secondUpdateTx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0d"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey(updateScriptHex(claimID)),
		}},
	}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{firstUpdateTx, purchaseTx, secondUpdateTx}, map[string]*lbrycrd.TxRawResult{
		firstUpdateTx.Txid:  firstUpdateTx,
		purchaseTx.Txid:     purchaseTx,
		secondUpdateTx.Txid: secondUpdateTx,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	if graph.unresolved[purchaseTx.Txid] != 1 {
		t.Fatalf("expected purchase to depend on first update, got %d dependencies", graph.unresolved[purchaseTx.Txid])
	}
	if graph.unresolved[secondUpdateTx.Txid] != 2 {
		t.Fatalf("expected second update to depend on first update and purchase, got %d dependencies", graph.unresolved[secondUpdateTx.Txid])
	}
}

func TestClaimGraphMalformedScriptDoesNotPanic(t *testing.T) {
	tx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0e"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey("b601"),
		}},
	}
	events := claimGraphEvents(tx)
	if len(events) != 0 {
		t.Fatalf("expected malformed script to produce no graph events, got %d", len(events))
	}
}

func TestClaimGraphMalformedClaimNameDoesNotInventWriter(t *testing.T) {
	tx := &lbrycrd.TxRawResult{
		Txid: hexTxID("0f"),
		Vout: []lbrycrd.Vout{{
			N:            0,
			ScriptPubKey: scriptPubKey("b5"),
		}},
	}
	events := claimGraphEvents(tx)
	if len(events) != 0 {
		t.Fatalf("expected malformed claim-name script to produce no graph events, got %d", len(events))
	}
}

func TestSchedulerErrorClassification(t *testing.T) {
	leftTx := &lbrycrd.TxRawResult{Txid: "left", Vin: []lbrycrd.Vin{{TxID: "right"}}}
	rightTx := &lbrycrd.TxRawResult{Txid: "right", Vin: []lbrycrd.Vin{{TxID: "left"}}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{leftTx, rightTx}, map[string]*lbrycrd.TxRawResult{
		"left":  leftTx,
		"right": rightTx,
	})
	err := graph.buildDependencies()
	if !errors.Is(err, ErrDependencyGraphStalled) {
		t.Fatalf("expected dependency graph sentinel, got %v", err)
	}

	result := txProcessResult{
		tx:          &lbrycrd.TxRawResult{Txid: "child"},
		err:         &MissingSourceOutputError{PrevoutTxID: "parent", PrevoutN: 0},
		blockHeight: 7,
	}
	parentTx := &lbrycrd.TxRawResult{Txid: "parent"}
	graph = newBlockTxGraph([]*lbrycrd.TxRawResult{parentTx, result.tx}, map[string]*lbrycrd.TxRawResult{
		"parent": parentTx,
		"child":  result.tx,
	})
	err = handleScheduledTxError(result, graph, nil, nil)
	if !errors.Is(err, ErrSchedulerInvariant) {
		t.Fatalf("expected scheduler invariant sentinel, got %v", err)
	}
}

func TestIndependentTransactionsRunInParallel(t *testing.T) {
	disableSchedulerCleanup(t)
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 2
	started := make(chan string, 2)
	release := make(chan struct{})
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		started <- tx.Txid
		<-release
		return nil
	}
	defer restoreSchedulerTestGlobals(originalProcessTx, originalMaxParallel)

	leftTx := &lbrycrd.TxRawResult{Txid: "left"}
	rightTx := &lbrycrd.TxRawResult{Txid: "right"}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{leftTx, rightTx}, map[string]*lbrycrd.TxRawResult{
		"left":  leftTx,
		"right": rightTx,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go runSchedulerForTest(done, graph)
	assertStarted(t, started)
	assertStarted(t, started)
	close(release)
	err = <-done
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlockTxGraphCycleDetection(t *testing.T) {
	leftTx := &lbrycrd.TxRawResult{Txid: "left", Vin: []lbrycrd.Vin{{TxID: "right"}}}
	rightTx := &lbrycrd.TxRawResult{Txid: "right", Vin: []lbrycrd.Vin{{TxID: "left"}}}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{leftTx, rightTx}, map[string]*lbrycrd.TxRawResult{
		"left":  leftTx,
		"right": rightTx,
	})
	err := graph.buildDependencies()
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestFetchBlockRawTransactionsPreservesOrder(t *testing.T) {
	originalFetchRawTransaction := fetchRawTransaction
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 2
	fetchRawTransaction = func(txID string) (*lbrycrd.TxRawResult, error) {
		return &lbrycrd.TxRawResult{Txid: txID}, nil
	}
	defer restoreSchedulerFetchTestGlobals(originalFetchRawTransaction, originalMaxParallel)

	orderedTxs, txByID, err := fetchBlockRawTransactions(stop.New(nil), []string{"b", "a"})
	if err != nil {
		t.Fatal(err)
	}
	if orderedTxs[0].Txid != "b" || orderedTxs[1].Txid != "a" {
		t.Fatalf("unexpected fetch order: %s, %s", orderedTxs[0].Txid, orderedTxs[1].Txid)
	}
	if txByID["a"] == nil || txByID["b"] == nil {
		t.Fatal("expected fetched tx map entries")
	}
}

func TestFetchBlockRawTransactionsPropagatesError(t *testing.T) {
	originalFetchRawTransaction := fetchRawTransaction
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 1
	fetchRawTransaction = func(txID string) (*lbrycrd.TxRawResult, error) {
		if txID == "bad" {
			return nil, errors.New("fetch failed")
		}
		return &lbrycrd.TxRawResult{Txid: txID}, nil
	}
	defer restoreSchedulerFetchTestGlobals(originalFetchRawTransaction, originalMaxParallel)

	_, _, err := fetchBlockRawTransactions(stop.New(nil), []string{"bad"})
	if err == nil {
		t.Fatal("expected fetch error")
	}
}

func TestFetchBlockRawTransactionsStopsOnCancellation(t *testing.T) {
	originalFetchRawTransaction := fetchRawTransaction
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 1
	started := make(chan string, 1)
	release := make(chan struct{})
	fetchRawTransaction = func(txID string) (*lbrycrd.TxRawResult, error) {
		started <- txID
		<-release
		return &lbrycrd.TxRawResult{Txid: txID}, nil
	}
	defer restoreSchedulerFetchTestGlobals(originalFetchRawTransaction, originalMaxParallel)

	stopper := stop.New(nil)
	done := make(chan error, 1)
	go runFetchForTest(done, stopper, []string{"first", "second"})
	assertStarted(t, started)
	stopper.Stop()
	time.Sleep(20 * time.Millisecond)
	close(release)
	err := <-done
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if err.Error() != ManualShutDownError.Error() {
		t.Fatalf("expected manual shutdown error, got %v", err)
	}
}

func TestRetryableStorageErrorClassification(t *testing.T) {
	if !isRetryableStorageError(&mysql.MySQLError{Number: mysqlErrorLockWaitTimeout}) {
		t.Fatal("expected lock wait timeout to be retryable")
	}
	if !isRetryableStorageError(&mysql.MySQLError{Number: mysqlErrorDeadlock}) {
		t.Fatal("expected deadlock to be retryable")
	}
	if isRetryableStorageError(&mysql.MySQLError{Number: 1062}) {
		t.Fatal("expected duplicate key to be non-retryable")
	}
}

func TestCleanupAbortedBlockTransactionsResetsSpentOutputs(t *testing.T) {
	testDB := newSQLBoilerTestDB(t)
	defer testDB.close(t)

	testDB.mock.ExpectQuery(selectFrom(model.TableNames.Input)).
		WithArgs("tx-a", "tx-b").
		WillReturnRows(inputRows(
			&model.Input{ID: 11, TransactionHash: "tx-a"},
			&model.Input{ID: 12, TransactionHash: "tx-b"},
		))
	testDB.mock.ExpectExec(updateOutputSpentByInput()).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	testDB.mock.ExpectQuery(countFrom(model.TableNames.Output)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := cleanupAbortedBlockTransactions([]string{"tx-a", "tx-b"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerDoesNotCleanupWhileWorkerIsActive(t *testing.T) {
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	originalCleanup := cleanupBlockTransactions
	MaxParallelTxProcessing = 2
	started := make(chan string, 2)
	release := make(chan struct{})
	cleanupCalled := make(chan struct{}, 1)
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		started <- tx.Txid
		if tx.Txid == "slow" {
			<-release
			return nil
		}
		return errors.New("terminal")
	}
	cleanupBlockTransactions = func(txIDs []string) error {
		cleanupCalled <- struct{}{}
		return nil
	}
	defer restoreSchedulerTestGlobals(originalProcessTx, originalMaxParallel)
	defer func() {
		cleanupBlockTransactions = originalCleanup
	}()

	slowTx := &lbrycrd.TxRawResult{Txid: "slow"}
	failTx := &lbrycrd.TxRawResult{Txid: "fail"}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{slowTx, failTx}, map[string]*lbrycrd.TxRawResult{
		"slow": slowTx,
		"fail": failTx,
	})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go runSchedulerForTest(done, graph)
	assertStarted(t, started)
	assertStarted(t, started)
	select {
	case <-cleanupCalled:
		t.Fatal("cleanup ran while worker was still active")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	err = <-done
	if err == nil {
		t.Fatal("expected terminal scheduler error")
	}
	select {
	case <-cleanupCalled:
	case <-time.After(time.Second):
		t.Fatal("expected cleanup after active worker completed")
	}
}

func TestSchedulerCancellationWaitsForActiveWorkerResult(t *testing.T) {
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	originalCleanup := cleanupBlockTransactions
	MaxParallelTxProcessing = 1
	started := make(chan string, 1)
	release := make(chan struct{})
	cleanupCalled := make(chan struct{}, 1)
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		started <- tx.Txid
		<-release
		return nil
	}
	cleanupBlockTransactions = func(txIDs []string) error {
		cleanupCalled <- struct{}{}
		return nil
	}
	defer restoreSchedulerTestGlobals(originalProcessTx, originalMaxParallel)
	defer func() {
		cleanupBlockTransactions = originalCleanup
	}()

	tx := &lbrycrd.TxRawResult{Txid: "tx"}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{tx}, map[string]*lbrycrd.TxRawResult{"tx": tx})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	stopper := stop.New(nil)
	done := make(chan error, 1)
	go runSchedulerWithStopperForTest(done, stopper, graph)
	assertStarted(t, started)
	stopper.Stop()
	close(release)
	select {
	case err = <-done:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not return after active worker completed")
	}
	if err == nil || err.Error() != ManualShutDownError.Error() {
		t.Fatalf("expected manual shutdown error, got %v", err)
	}
	select {
	case <-cleanupCalled:
	case <-time.After(time.Second):
		t.Fatal("expected cleanup after active worker completed")
	}
}

func TestSchedulerDoesNotDispatchAfterStop(t *testing.T) {
	disableSchedulerCleanup(t)
	originalProcessTx := processTx
	originalMaxParallel := MaxParallelTxProcessing
	MaxParallelTxProcessing = 1
	dispatched := false
	processTx = func(tx *lbrycrd.TxRawResult, blockTime uint64, blockHeight uint64) error {
		dispatched = true
		return nil
	}
	defer restoreSchedulerTestGlobals(originalProcessTx, originalMaxParallel)

	tx := &lbrycrd.TxRawResult{Txid: "tx"}
	graph := newBlockTxGraph([]*lbrycrd.TxRawResult{tx}, map[string]*lbrycrd.TxRawResult{"tx": tx})
	err := graph.buildDependencies()
	if err != nil {
		t.Fatal(err)
	}
	stopper := stop.New(nil)
	stopper.Stop()
	err = runBlockTxScheduler(stopper, graph, 1, 2)
	if err == nil || err.Error() != ManualShutDownError.Error() {
		t.Fatalf("expected manual shutdown error, got %v", err)
	}
	if dispatched {
		t.Fatal("transaction dispatched after stopper was stopped")
	}
}

func restoreSchedulerTestGlobals(originalProcessTx func(*lbrycrd.TxRawResult, uint64, uint64) error, maxParallel int) {
	processTx = originalProcessTx
	MaxParallelTxProcessing = maxParallel
}

func restoreSchedulerRetryTestGlobals(originalProcessTx func(*lbrycrd.TxRawResult, uint64, uint64) error, maxParallel int, maxFailures int, retryBackoff time.Duration) {
	processTx = originalProcessTx
	MaxParallelTxProcessing = maxParallel
	MaxFailures = maxFailures
	txRetryBackoff = retryBackoff
}

func restoreSchedulerFetchTestGlobals(originalFetchRawTransaction func(string) (*lbrycrd.TxRawResult, error), maxParallel int) {
	fetchRawTransaction = originalFetchRawTransaction
	MaxParallelTxProcessing = maxParallel
}

func restoreSchedulerEndToEndTestGlobals(originalFetchRawTransaction func(string) (*lbrycrd.TxRawResult, error), originalProcessTx func(*lbrycrd.TxRawResult, uint64, uint64) error, maxParallel int) {
	fetchRawTransaction = originalFetchRawTransaction
	processTx = originalProcessTx
	MaxParallelTxProcessing = maxParallel
}

func runSchedulerForTest(done chan<- error, graph *blockTxGraph) {
	done <- runBlockTxScheduler(stop.New(nil), graph, 1, 2)
}

func runSchedulerWithStopperForTest(done chan<- error, stopper *stop.Group, graph *blockTxGraph) {
	done <- runBlockTxScheduler(stopper, graph, 1, 2)
}

func runFetchForTest(done chan<- error, stopper *stop.Group, txIDs []string) {
	_, _, err := fetchBlockRawTransactions(stopper, txIDs)
	done <- err
}

func assertStarted(t *testing.T, started <-chan string) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for transaction to start")
	}
}

func hexTxID(suffix string) string {
	return strings.Repeat("0", 64-len(suffix)) + suffix
}

func scriptPubKey(hex string) btcjson.ScriptPubKeyResult {
	return btcjson.ScriptPubKeyResult{Hex: hex}
}

func claimNameScriptHex() string {
	return "b50178006d75"
}

func supportScriptHex(claimID string) string {
	claimBytes, _ := hex.DecodeString(claimID)
	for left, right := 0, len(claimBytes)-1; left < right; left, right = left+1, right-1 {
		claimBytes[left], claimBytes[right] = claimBytes[right], claimBytes[left]
	}
	return "b6016114" + hex.EncodeToString(claimBytes) + "6d75"
}

func updateScriptHex(claimID string) string {
	claimBytes, _ := hex.DecodeString(claimID)
	for left, right := 0, len(claimBytes)-1; left < right; left, right = left+1, right-1 {
		claimBytes[left], claimBytes[right] = claimBytes[right], claimBytes[left]
	}
	return "b7017814" + hex.EncodeToString(claimBytes) + "006d75"
}

func purchaseScriptHex(t *testing.T, claimID string) string {
	t.Helper()
	claimBytes, err := hex.DecodeString(claimID)
	if err != nil {
		t.Fatal(err)
	}
	for left, right := 0, len(claimBytes)-1; left < right; left, right = left+1, right-1 {
		claimBytes[left], claimBytes[right] = claimBytes[right], claimBytes[left]
	}
	purchaseBytes, err := proto.Marshal(&pb.Purchase{ClaimHash: claimBytes})
	if err != nil {
		t.Fatal(err)
	}
	payload := append([]byte{0x50}, purchaseBytes...)
	script := append([]byte{0x6a, byte(len(payload))}, payload...)
	return hex.EncodeToString(script)
}

func inputRows(inputs ...*model.Input) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		model.InputColumns.ID,
		model.InputColumns.TransactionID,
		model.InputColumns.TransactionHash,
		model.InputColumns.InputAddressID,
		model.InputColumns.IsCoinbase,
		model.InputColumns.Coinbase,
		model.InputColumns.PrevoutHash,
		model.InputColumns.PrevoutN,
		model.InputColumns.Sequence,
		model.InputColumns.Value,
		model.InputColumns.ScriptSigAsm,
		model.InputColumns.ScriptSigHex,
		model.InputColumns.Created,
		model.InputColumns.Modified,
		model.InputColumns.Vin,
		model.InputColumns.Witness,
	})
	for _, input := range inputs {
		rows.AddRow(
			int64(input.ID),
			int64(input.TransactionID),
			input.TransactionHash,
			nullUint64Value(input.InputAddressID),
			input.IsCoinbase,
			nullStringValue(input.Coinbase),
			nullStringValue(input.PrevoutHash),
			nullUintValue(input.PrevoutN),
			int64(input.Sequence),
			nullFloat64Value(input.Value),
			nullStringValue(input.ScriptSigAsm),
			nullStringValue(input.ScriptSigHex),
			input.Created,
			input.Modified,
			nullUintValue(input.Vin),
			nullStringValue(input.Witness),
		)
	}
	return rows
}

func nullUintValue(value null.Uint) interface{} {
	if !value.Valid {
		return nil
	}
	return int64(value.Uint)
}

func nullFloat64Value(value null.Float64) interface{} {
	if !value.Valid {
		return nil
	}
	return value.Float64
}

func updateOutputSpentByInput() string {
	query := "UPDATE `" + model.TableNames.Output + "` SET `" + model.OutputColumns.IsSpent + "` = ?, `" + model.OutputColumns.SpentByInputID + "` = ? WHERE"
	return regexp.QuoteMeta(query)
}

func disableSchedulerCleanup(t *testing.T) {
	originalCleanup := cleanupBlockTransactions
	cleanupBlockTransactions = func(txIDs []string) error {
		return nil
	}
	t.Cleanup(func() {
		cleanupBlockTransactions = originalCleanup
	})
}
