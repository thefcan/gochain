package wallet

import (
	"path/filepath"
	"testing"
)

func TestWalletsSaveAndLoad(t *testing.T) {
	file := filepath.Join(t.TempDir(), "wallet.dat")

	ws, err := LoadWallets(file) // missing file -> empty set
	if err != nil {
		t.Fatalf("LoadWallets: %v", err)
	}
	addr, err := ws.CreateWallet()
	if err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}
	if err := ws.Save(file); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadWallets(file)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	w, ok := loaded.GetWallet(addr)
	if !ok {
		t.Fatalf("address %q not found after reload", addr)
	}
	if w.Address() != addr {
		t.Errorf("reconstructed address = %q, want %q", w.Address(), addr)
	}
	// The private key must round-trip (modern API, not the deprecated D field).
	raw, err := w.PrivateKey.Bytes()
	if err != nil || len(raw) == 0 {
		t.Errorf("private key was not restored: %v", err)
	}
}

func TestLoadWalletsMissingFileIsEmpty(t *testing.T) {
	ws, err := LoadWallets(filepath.Join(t.TempDir(), "nope.dat"))
	if err != nil {
		t.Fatalf("LoadWallets on missing file: %v", err)
	}
	if n := len(ws.GetAddresses()); n != 0 {
		t.Errorf("addresses = %d, want 0", n)
	}
}
