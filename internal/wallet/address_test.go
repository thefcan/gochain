package wallet

import "testing"

func TestPubKeyHashFromAddress(t *testing.T) {
	w, _ := NewWallet()
	ph, err := PubKeyHashFromAddress(w.Address())
	if err != nil {
		t.Fatalf("PubKeyHashFromAddress: %v", err)
	}
	if len(ph) != 20 {
		t.Errorf("pubKeyHash length = %d, want 20", len(ph))
	}
	if string(ph) != string(HashPubKey(w.PublicKey)) {
		t.Error("extracted pubKeyHash != HashPubKey(public key)")
	}
}

func TestPubKeyHashFromAddressRejectsGarbage(t *testing.T) {
	if _, err := PubKeyHashFromAddress("!!!"); err == nil {
		t.Error("expected error for an invalid address")
	}
}
