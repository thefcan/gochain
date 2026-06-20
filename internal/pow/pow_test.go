package pow

import (
	"math/big"
	"testing"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/tx"
)

func mineableBlock(t *testing.T, data string) *block.Block {
	t.Helper()
	cb, err := tx.NewCoinbaseTX("tester", data)
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	return block.New([]*tx.Transaction{cb}, []byte{})
}

func TestRunProducesValidHash(t *testing.T) {
	b := mineableBlock(t, "mine me")
	p := New(b)

	nonce, hash := p.Run()
	b.Nonce, b.Hash = nonce, hash

	if len(hash) != 32 {
		t.Fatalf("hash length = %d, want 32", len(hash))
	}
	var hashInt big.Int
	hashInt.SetBytes(hash)
	if hashInt.Cmp(p.target) != -1 {
		t.Error("mined hash is not below the target")
	}
	if !p.Validate() {
		t.Error("Validate() = false for a freshly mined block")
	}
}

func TestValidateRejectsTamperedBlock(t *testing.T) {
	b := mineableBlock(t, "honest")
	nonce, hash := New(b).Run()
	b.Nonce, b.Hash = nonce, hash

	// Tampering with the transaction set must invalidate the proof.
	b.Transactions[0].ID = []byte("tampered")
	if New(b).Validate() {
		t.Error("Validate() = true for a tampered block; want false")
	}
}

func BenchmarkRun(b *testing.B) {
	cb, _ := tx.NewCoinbaseTX("bench", "x")
	for i := 0; i < b.N; i++ {
		blk := block.New([]*tx.Transaction{cb}, []byte{})
		New(blk).Run()
	}
}
