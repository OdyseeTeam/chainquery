package jobs

import (
	"errors"
	"fmt"
	"testing"
)

func TestChainSyncRecordAndReturnErrorDedupes(t *testing.T) {
	status := &chainSyncStatus{}
	err := errors.New("boom")

	status.recordError(1, "area", err)
	status.recordError(2, "area", err)

	if len(status.Errors) != 1 {
		t.Fatalf("expected one deduped error, got %d", len(status.Errors))
	}
	if len(status.Errors[0].HeightFound) != 2 {
		t.Fatalf("expected both heights to be retained, got %v", status.Errors[0].HeightFound)
	}
}

func TestChainSyncRecordAndReturnErrorCapsStoredErrors(t *testing.T) {
	status := &chainSyncStatus{}
	for i := 0; i < maxChainSyncErrors+5; i++ {
		status.recordError(int64(i), "area", fmt.Errorf("err%d", i))
	}

	if len(status.Errors) != maxChainSyncErrors {
		t.Fatalf("expected %d errors, got %d", maxChainSyncErrors, len(status.Errors))
	}
	if status.Errors[0].HeightFound[0] != 5 {
		t.Fatalf("expected oldest retained height to be 5, got %d", status.Errors[0].HeightFound[0])
	}
}
