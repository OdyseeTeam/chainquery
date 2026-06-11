package daemon

import (
	"testing"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/stop"
)

type blockWaitResult struct {
	height uint64
	ok     bool
}

func TestWaitForBlockProcessedContinuesAfterTimeout(t *testing.T) {
	oldStopper := stopper
	oldBlockProcessedChan := blockProcessedChan
	oldTimeout := blockProcessingTimeout
	oldDumpDelay := blockProcessingDumpDelay
	oldExit := exitOnBlockProcessingTimeout
	defer func() {
		stopper = oldStopper
		blockProcessedChan = oldBlockProcessedChan
		blockProcessingTimeout = oldTimeout
		blockProcessingDumpDelay = oldDumpDelay
		exitOnBlockProcessingTimeout = oldExit
		watchdogState = blockWatchdogState{}
	}()

	stopper = stop.New()
	blockProcessedChan = make(chan uint64)
	blockProcessingTimeout = 10 * time.Millisecond
	blockProcessingDumpDelay = 10 * time.Millisecond
	exitOnBlockProcessingTimeout = false
	recordDaemonTarget(9)
	recordBlockStart(7)

	results := make(chan blockWaitResult, 1)
	go waitForBlockProcessedForTest(7, results)
	time.Sleep(25 * time.Millisecond)
	blockProcessedChan <- 6

	select {
	case result := <-results:
		if !result.ok {
			t.Fatal("expected wait to complete after delayed block result")
		}
		if result.height != 6 {
			t.Fatalf("expected processed height 6, got %d", result.height)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for block processed result")
	}
}

func waitForBlockProcessedForTest(height uint64, results chan<- blockWaitResult) {
	processedHeight, ok := waitForBlockProcessed(height)
	results <- blockWaitResult{height: processedHeight, ok: ok}
}
