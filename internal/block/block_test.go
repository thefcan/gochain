package block

import (
	"testing"

	"github.com/thefcan/gochain/internal/tx"
	"github.com/thefcan/gochain/internal/wallet"
)

func coinbase(t *testing.T, addr, data string) *tx.Transaction {
	t.Helper()
	cb, err := tx.NewCoinbaseTX(addr, data)
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	return cb
}

func TestNewReturnsUnminedBlock(t *testing.T) {
	w, _ := wallet.NewWallet()
	b := New([]*tx.Transaction{coinbase(t, w.Address(), "data")}, []byte("prevhash"))

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
	w, _ := wallet.NewWallet()
	addr := w.Address()
	a := New([]*tx.Transaction{coinbase(t, addr, "same")}, []byte{})
	b := New([]*tx.Transaction{coinbase(t, addr, "same")}, []byte{})
	c := New([]*tx.Transaction{coinbase(t, addr, "different")}, []byte{})

	if string(a.HashTransactions()) != string(b.HashTransactions()) {
		t.Error("identical transactions should hash equally")
	}
	if string(a.HashTransactions()) == string(c.HashTransactions()) {
		t.Error("different transactions should hash differently")
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	w, _ := wallet.NewWallet()
	orig := New([]*tx.Transaction{coinbase(t, w.Address(), "data")}, []byte("prev"))
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
	if len(got.Transactions[0].Vout[0].PubKeyHash) != 20 {
		t.Errorf("output PubKeyHash not preserved: %+v", got.Transactions[0].Vout[0])
	}
}
