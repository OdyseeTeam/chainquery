package processing

import (
	stderrors "errors"
	"fmt"
)

var (
	ErrDependencyGraphStalled = stderrors.New("dependency graph stalled")
	ErrSchedulerInvariant     = stderrors.New("scheduler invariant violation")
)

type MissingSourceOutputError struct {
	PrevoutTxID string
	TxID        string
	PrevoutN    uint
	BlockHeight uint64
}

func (err *MissingSourceOutputError) Error() string {
	if err.TxID != "" || err.BlockHeight > 0 {
		return fmt.Sprintf("missing source output for %s:%d while processing tx %s in block %d", err.PrevoutTxID, err.PrevoutN, err.TxID, err.BlockHeight)
	}
	return fmt.Sprintf("Missing source output for %s:%d", err.PrevoutTxID, err.PrevoutN)
}

func enrichMissingSourceOutput(err error, txID string, blockHeight uint64) error {
	var missing *MissingSourceOutputError
	if !stderrors.As(err, &missing) {
		return err
	}
	enriched := *missing
	enriched.TxID = txID
	enriched.BlockHeight = blockHeight
	return &enriched
}

func missingSourceOutputFromError(err error) (*MissingSourceOutputError, bool) {
	var missing *MissingSourceOutputError
	if stderrors.As(err, &missing) {
		return missing, true
	}
	return nil, false
}
