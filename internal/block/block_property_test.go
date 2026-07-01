package block_test

import (
	"bytes"
	"testing"

	"pgregory.net/rapid"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/tx"
)

func drawTx(rt *rapid.T) *tx.Transaction {
	nIn := rapid.IntRange(0, 3).Draw(rt, "nIn")
	nOut := rapid.IntRange(1, 3).Draw(rt, "nOut")

	vin := make([]tx.TXInput, nIn)
	for i := range vin {
		vin[i] = tx.TXInput{
			Txid:      rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(rt, "txid"),
			Vout:      rapid.IntRange(0, 10).Draw(rt, "vout"),
			Signature: rapid.SliceOfN(rapid.Byte(), 1, 64).Draw(rt, "sig"),
			PubKey:    rapid.SliceOfN(rapid.Byte(), 1, 65).Draw(rt, "pubkey"),
		}
	}
	vout := make([]tx.TXOutput, nOut)
	for i := range vout {
		vout[i] = tx.TXOutput{
			Value:      rapid.IntRange(0, 1_000_000).Draw(rt, "value"),
			PubKeyHash: rapid.SliceOfN(rapid.Byte(), 1, 20).Draw(rt, "pubkeyhash"),
		}
	}
	return &tx.Transaction{
		ID:   rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(rt, "id"),
		Vin:  vin,
		Vout: vout,
	}
}

func drawBlock(rt *rapid.T) *block.Block {
	n := rapid.IntRange(1, 3).Draw(rt, "nTx")
	txs := make([]*tx.Transaction, n)
	for i := range txs {
		txs[i] = drawTx(rt)
	}
	return &block.Block{
		Timestamp:     rapid.Int64().Draw(rt, "timestamp"),
		Transactions:  txs,
		PrevBlockHash: rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(rt, "prev"),
		Hash:          rapid.SliceOfN(rapid.Byte(), 1, 32).Draw(rt, "hash"),
		Nonce:         rapid.IntRange(0, 1_000_000).Draw(rt, "nonce"),
	}
}

// TestSerializeRoundTripIdempotent asserts that once a block has been through
// one encode/decode cycle (which canonicalises nil vs empty slices), every
// further cycle is byte-for-byte stable.
func TestSerializeRoundTripIdempotent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := drawBlock(rt)

		s1, err := b.Serialize()
		if err != nil {
			rt.Fatalf("Serialize: %v", err)
		}
		b2, err := block.Deserialize(s1)
		if err != nil {
			rt.Fatalf("Deserialize: %v", err)
		}
		s2, err := b2.Serialize()
		if err != nil {
			rt.Fatalf("re-Serialize: %v", err)
		}
		b3, err := block.Deserialize(s2)
		if err != nil {
			rt.Fatalf("Deserialize (2): %v", err)
		}
		s3, err := b3.Serialize()
		if err != nil {
			rt.Fatalf("re-Serialize (2): %v", err)
		}
		if !bytes.Equal(s2, s3) {
			rt.Fatalf("serialize not idempotent:\n s2=%x\n s3=%x", s2, s3)
		}
	})
}

// TestHashTransactionsDeterministicAndSensitive asserts that the transaction
// root is a pure function of the transaction IDs and changes whenever any ID
// changes.
func TestHashTransactionsDeterministicAndSensitive(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		b := drawBlock(rt)

		h1 := b.HashTransactions()
		h2 := b.HashTransactions()
		if !bytes.Equal(h1, h2) {
			rt.Fatal("HashTransactions not deterministic")
		}

		original := b.Transactions[0].ID
		b.Transactions[0].ID = append(append([]byte(nil), original...), 0x01)
		h3 := b.HashTransactions()
		if bytes.Equal(h1, h3) {
			rt.Fatal("HashTransactions is insensitive to a transaction ID change")
		}
	})
}
