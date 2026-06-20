package block

import (
	"testing"

	"github.com/thefcan/gochain/internal/tx"
)

func coinbase(t *testing.T, data string) *tx.Transaction {
	t.Helper()
	cb, err := tx.NewCoinbaseTX("tester", data)
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	return cb
}

func TestNewReturnsUnminedBlock(t *testing.T) {
	b := New([]*tx.Transaction{coinbase(t, "data")}, []byte("prevhash"))

	if len(b.Transactions) != 1 {
		t.Errorf("Transactions len = %d, want 1", len(b.Transactions))
	}
	if string(b.PrevBlockHash) != "prevhash" {
		t.Errorf("PrevBlockHash = %q, want prevhash", b.PrevBlockHash)
	}
	if b.Timestamp == 0 {
		t.Error("Timestamp was not set")
	}
	if b.Hash != nil {
		t.Errorf("Hash = %x, want nil before mining", b.Hash)
	}
}

func TestHashTransactions(t *testing.T) {
	a := New([]*tx.Transaction{coinbase(t, "same")}, []byte{})
	b := New([]*tx.Transaction{coinbase(t, "same")}, []byte{})
	c := New([]*tx.Transaction{coinbase(t, "different")}, []byte{})

	if string(a.HashTransactions()) != string(b.HashTransactions()) {
		t.Error("identical transactions should hash equally")
	}
	if string(a.HashTransactions()) == string(c.HashTransactions()) {
		t.Error("different transactions should hash differently")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	orig := New([]*tx.Transaction{coinbase(t, "data")}, []byte("prev"))
	orig.Hash = []byte("somehash")
	orig.Nonce = 42

	data, err := orig.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	got, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if got.Nonce != 42 || string(got.Hash) != "somehash" || len(got.Transactions) != 1 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.Transactions[0].Vout[0].ScriptPubKey != "tester" {
		t.Errorf("transaction not preserved: %+v", got.Transactions[0])
	}
}
