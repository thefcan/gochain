package tx_test

import (
	"testing"

	"github.com/thefcan/gochain/internal/tx"
)

// TestVerifyOutOfRangeVoutNoPanic asserts that an input referencing an
// out-of-range (here negative) output index is rejected cleanly instead of
// panicking the verifier — a crafted transaction must never crash a node.
// Before the bounds fix this test panicked with an index-out-of-range.
func TestVerifyOutOfRangeVoutNoPanic(t *testing.T) {
	prev := &tx.Transaction{
		ID:   []byte{0x01},
		Vout: []tx.TXOutput{{Value: 5, PubKeyHash: []byte{0x02}}},
	}
	prevTXs := map[string]tx.Transaction{
		"01": *prev,
	}
	spend := &tx.Transaction{
		ID: []byte{0x09},
		Vin: []tx.TXInput{{
			Txid:      prev.ID,
			Vout:      -1, // out of range
			Signature: make([]byte, 64),
			PubKey:    make([]byte, 65),
		}},
		Vout: []tx.TXOutput{{Value: 5, PubKeyHash: []byte{0x02}}},
	}

	ok, err := spend.Verify(prevTXs)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if ok {
		t.Fatal("a transaction with an out-of-range Vout verified as valid")
	}
}
