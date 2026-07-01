package tx_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"pgregory.net/rapid"

	"github.com/thefcan/gochain/internal/tx"
	"github.com/thefcan/gochain/internal/wallet"
)

// drawTx builds a structurally random transaction. Its contents need not be
// semantically valid: the properties here concern encoding and hashing, not
// consensus rules.
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

// TestHashDeterministicAndIDIndependent asserts that Hash is a pure function of
// the transaction contents and, because Hash clears the ID before encoding, is
// independent of the ID field.
func TestHashDeterministicAndIDIndependent(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		txn := drawTx(rt)

		h1, err := txn.Hash()
		if err != nil {
			rt.Fatalf("Hash: %v", err)
		}
		h2, err := txn.Hash()
		if err != nil {
			rt.Fatalf("Hash (2nd call): %v", err)
		}
		if !bytes.Equal(h1, h2) {
			rt.Fatalf("Hash not deterministic: %x != %x", h1, h2)
		}

		clone := *txn
		clone.ID = []byte("a-completely-different-id")
		h3, err := clone.Hash()
		if err != nil {
			rt.Fatalf("Hash (clone): %v", err)
		}
		if !bytes.Equal(h1, h3) {
			rt.Fatalf("Hash depends on ID field: %x != %x", h1, h3)
		}
	})
}

// TestSignVerifySoundness asserts that a correctly signed input verifies, and
// that tampering with either the signature or a signed output makes
// verification fail.
func TestSignVerifySoundness(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		w, err := wallet.NewWallet()
		if err != nil {
			rt.Fatalf("NewWallet: %v", err)
		}
		pkh := wallet.HashPubKey(w.PublicKey)

		// A previous transaction with a single output at index 0.
		prev := &tx.Transaction{
			Vout: []tx.TXOutput{{
				Value:      rapid.IntRange(1, 1_000).Draw(rt, "prevValue"),
				PubKeyHash: pkh,
			}},
		}
		if prev.ID, err = prev.Hash(); err != nil {
			rt.Fatalf("prev.Hash: %v", err)
		}

		outValue := rapid.IntRange(1, 1_000).Draw(rt, "outValue")
		spend := &tx.Transaction{
			Vin:  []tx.TXInput{{Txid: prev.ID, Vout: 0, PubKey: w.PublicKey}},
			Vout: []tx.TXOutput{{Value: outValue, PubKeyHash: pkh}},
		}
		if spend.ID, err = spend.Hash(); err != nil {
			rt.Fatalf("spend.Hash: %v", err)
		}

		prevTXs := map[string]tx.Transaction{
			hex.EncodeToString(prev.ID): *prev,
		}

		if err := spend.Sign(w.PrivateKey, prevTXs); err != nil {
			rt.Fatalf("Sign: %v", err)
		}

		ok, err := spend.Verify(prevTXs)
		if err != nil {
			rt.Fatalf("Verify (valid): %v", err)
		}
		if !ok {
			rt.Fatal("a correctly signed transaction failed to verify")
		}

		// Tamper 1: flip a bit in the signature (length preserved).
		flipped := append([]byte(nil), spend.Vin[0].Signature...)
		flipped[0] ^= 0xFF
		bad1 := &tx.Transaction{
			ID:   spend.ID,
			Vin:  []tx.TXInput{{Txid: prev.ID, Vout: 0, Signature: flipped, PubKey: w.PublicKey}},
			Vout: spend.Vout,
		}
		ok, err = bad1.Verify(prevTXs)
		if err != nil {
			rt.Fatalf("Verify (flipped sig): %v", err)
		}
		if ok {
			rt.Fatal("a transaction with a flipped signature bit verified as valid")
		}

		// Tamper 2: mutate a signed output value after signing.
		bad2 := &tx.Transaction{
			ID:   spend.ID,
			Vin:  []tx.TXInput{{Txid: prev.ID, Vout: 0, Signature: spend.Vin[0].Signature, PubKey: w.PublicKey}},
			Vout: []tx.TXOutput{{Value: outValue + 1, PubKeyHash: pkh}},
		}
		ok, err = bad2.Verify(prevTXs)
		if err != nil {
			rt.Fatalf("Verify (mutated output): %v", err)
		}
		if ok {
			rt.Fatal("a transaction with a mutated output verified as valid")
		}
	})
}
