package pow

import (
	"math/big"
	"testing"

	"github.com/thefcan/gochain/internal/block"
)

func TestRunProducesValidHash(t *testing.T) {
	b := block.New("mine me", []byte{})
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
	b := block.New("honest", []byte{})
	nonce, hash := New(b).Run()
	b.Nonce, b.Hash = nonce, hash

	// Tampering with the data after mining must invalidate the proof.
	b.Data = []byte("tampered")
	if New(b).Validate() {
		t.Error("Validate() = true for a tampered block; want false")
	}
}

func BenchmarkRun(b *testing.B) {
	for i := 0; i < b.N; i++ {
		blk := block.New("benchmark", []byte{})
		New(blk).Run()
	}
}
