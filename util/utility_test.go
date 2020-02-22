package util

import (
	"encoding/hex"
	"testing"
)

func TestReverseBytes(t *testing.T) {
	originalHex := "ad779413d8710f52a5c1d79af7e10a329147576f"
	bytes, err := hex.DecodeString(originalHex)
	if err != nil {
		t.Error(err)
	}
	reversed := ReverseBytes(bytes)
	reversedHex := hex.EncodeToString(reversed)
	if reversedHex != "6f574791320ae1f79ad7c1a5520f71d8139477ad" {
		t.Error("it doesnt match")
	}
}
