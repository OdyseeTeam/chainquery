package processing

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/v2/extras/stop"
)

type vinProcessRecorder struct {
	count atomic.Int32
}

func (recorder *vinProcessRecorder) process(jsonVin *lbrycrd.Vin, tx *model.Transaction, txDC *txDebitCredits, n uint64) error {
	recorder.count.Add(1)
	if n == 1 {
		return errors.New("vin failure")
	}
	return nil
}

func TestSaveUpdateInputsDrainsAllVinResultsAfterError(t *testing.T) {
	originalProcessVin := processVin
	originalMaxParallel := MaxParallelVinProcessing
	recorder := &vinProcessRecorder{}
	processVin = recorder.process
	MaxParallelVinProcessing = 3
	defer restoreVinTestGlobals(originalProcessVin, originalMaxParallel)

	transaction := &model.Transaction{ID: 1, Hash: "tx"}
	jsonTx := &lbrycrd.TxRawResult{
		Txid: "tx",
		Vin: []lbrycrd.Vin{
			{Sequence: 1},
			{Sequence: 2},
			{Sequence: 3},
		},
	}

	err := saveUpdateInputs(transaction, jsonTx, newTxDebitCredits())
	if err == nil {
		t.Fatal("expected vin error")
	}
	if recorder.count.Load() != int32(len(jsonTx.Vin)) {
		t.Fatalf("expected all vins to be processed, got %d", recorder.count.Load())
	}
}

func restoreVinTestGlobals(originalProcessVin func(*lbrycrd.Vin, *model.Transaction, *txDebitCredits, uint64) error, maxParallel int) {
	processVin = originalProcessVin
	MaxParallelVinProcessing = maxParallel
}

func TestReprocessQueueStopsDuringTeardown(t *testing.T) {
	syncStopper := stop.New(nil)
	manager := &txSyncManager{
		syncStopper: syncStopper,
		redoJobsCh:  make(chan txToProcess, 1),
		jobsCh:      make(chan txToProcess),
	}
	manager.redoJobsCh <- txToProcess{tx: &lbrycrd.TxRawResult{Txid: "retry"}}
	syncStopper.Add(1)
	go reprocessQueue(manager)

	time.Sleep(10 * time.Millisecond)
	syncStopper.Stop()

	done := make(chan struct{})
	go waitForStopGroup(syncStopper, done)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reprocess queue teardown")
	}
}

func waitForStopGroup(group *stop.Group, done chan<- struct{}) {
	group.StopAndWait()
	close(done)
}
