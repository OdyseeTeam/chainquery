package processing

import (
	"testing"

	"github.com/lbryio/chainquery/lbrycrd"
)

func TestOptimizeOrderToProcessTopological(t *testing.T) {
	parent := &lbrycrd.TxRawResult{Txid: "parent"}
	child := &lbrycrd.TxRawResult{Txid: "child", Vin: []lbrycrd.Vin{{TxID: "parent"}}}
	grandchild := &lbrycrd.TxRawResult{Txid: "grandchild", Vin: []lbrycrd.Vin{{TxID: "child"}}}

	ordered, ok := optimizeOrderToProcess(map[string]*lbrycrd.TxRawResult{
		"grandchild": grandchild,
		"child":      child,
		"parent":     parent,
	})
	if !ok {
		t.Fatal("expected dependency ordering to succeed")
	}
	got := []string{ordered[0].Txid, ordered[1].Txid, ordered[2].Txid}
	want := []string{"parent", "child", "grandchild"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestOptimizeOrderToProcessCycleFallsBack(t *testing.T) {
	first := &lbrycrd.TxRawResult{Txid: "a", Vin: []lbrycrd.Vin{{TxID: "b"}}}
	second := &lbrycrd.TxRawResult{Txid: "b", Vin: []lbrycrd.Vin{{TxID: "a"}}}

	ordered, ok := optimizeOrderToProcess(map[string]*lbrycrd.TxRawResult{
		"b": second,
		"a": first,
	})
	if ok {
		t.Fatal("expected cycle detection to fail")
	}
	got := []string{ordered[0].Txid, ordered[1].Txid}
	want := []string{"a", "b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected deterministic fallback %v, got %v", want, got)
		}
	}
}
