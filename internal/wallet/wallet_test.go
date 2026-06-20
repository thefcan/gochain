package wallet

import (
	"bytes"
	"testing"
)

func TestNewWalletAddressIsValid(t *testing.T) {
	w, err := NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	if got := len(w.PublicKey); got != 2*fieldSize {
		t.Errorf("PublicKey length = %d, want %d", got, 2*fieldSize)
	}
	if !ValidateAddress(w.Address()) {
		t.Errorf("generated address %q failed validation", w.Address())
	}
}

func TestValidateAddressRejectsTampered(t *testing.T) {
	w, _ := NewWallet()
	bad := []byte(w.Address())
	if bad[len(bad)-1] == 'A' {
		bad[len(bad)-1] = 'B'
	} else {
		bad[len(bad)-1] = 'A'
	}
	if ValidateAddress(string(bad)) {
		t.Error("tampered address passed validation")
	}
}

func TestHashPubKeyLength(t *testing.T) {
	if got := len(HashPubKey([]byte("somekey"))); got != 20 {
		t.Errorf("HashPubKey length = %d, want 20 (RIPEMD160)", got)
	}
}

func TestBase58RoundTrip(t *testing.T) {
	for _, in := range [][]byte{{0x00, 0x01, 0x02}, []byte("hello"), {0x00, 0x00, 0xff, 0x10}} {
		if out := Base58Decode(Base58Encode(in)); !bytes.Equal(out, in) {
			t.Errorf("round-trip: got %x, want %x", out, in)
		}
	}
}
