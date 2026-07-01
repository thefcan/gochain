package tx

import (
	"encoding/hex"
	"testing"

	"github.com/thefcan/gochain/internal/wallet"
)

func TestCoinbaseLockedToRecipient(t *testing.T) {
	w, _ := wallet.NewWallet()
	cb, err := NewCoinbaseTX(w.Address(), "")
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	if !cb.IsCoinbase() {
		t.Error("IsCoinbase() = false, want true")
	}
	ph, _ := wallet.PubKeyHashFromAddress(w.Address())
	if !cb.Vout[0].IsLockedWithKey(ph) {
		t.Error("coinbase output not locked to its recipient")
	}
}

// spendFrom builds an unsigned tx where `owner` spends prev's output #0.
func spendFrom(t *testing.T, prev *Transaction, owner *wallet.Wallet, to string) *Transaction {
	t.Helper()
	out, err := NewTXOutput(Subsidy, to)
	if err != nil {
		t.Fatalf("NewTXOutput: %v", err)
	}
	spend := &Transaction{
		Vin:  []TXInput{{Txid: prev.ID, Vout: 0, PubKey: owner.PublicKey}},
		Vout: []TXOutput{*out},
	}
	id, err := spend.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	spend.ID = id
	return spend
}

func TestSignAndVerify(t *testing.T) {
	alice, _ := wallet.NewWallet()
	bob, _ := wallet.NewWallet()

	prev, _ := NewCoinbaseTX(alice.Address(), "genesis") // pays Alice
	prevTXs := map[string]Transaction{hex.EncodeToString(prev.ID): *prev}

	spend := spendFrom(t, prev, alice, bob.Address())
	if err := spend.Sign(alice.PrivateKey, prevTXs); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, err := spend.Verify(prevTXs)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Error("a valid signature failed verification")
	}
}

func TestVerifyRejectsForgedSignature(t *testing.T) {
	alice, _ := wallet.NewWallet()
	mallory, _ := wallet.NewWallet()
	bob, _ := wallet.NewWallet()

	prev, _ := NewCoinbaseTX(alice.Address(), "genesis") // pays Alice
	prevTXs := map[string]Transaction{hex.EncodeToString(prev.ID): *prev}

	// Mallory claims to be Alice (uses her pubkey) but signs with her own key.
	spend := spendFrom(t, prev, alice, bob.Address())
	if err := spend.Sign(mallory.PrivateKey, prevTXs); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	ok, _ := spend.Verify(prevTXs)
	if ok {
		t.Error("a forged signature passed verification")
	}
}
