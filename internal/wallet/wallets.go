package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/gob"
	"errors"
	"math/big"
	"os"
	"sort"
)

// Wallets is a collection of wallets keyed by their address.
type Wallets struct {
	wallets map[string]*Wallet
}

// LoadWallets reads wallets from file; a missing file yields an empty set.
func LoadWallets(file string) (*Wallets, error) {
	ws := &Wallets{wallets: make(map[string]*Wallet)}

	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return ws, nil
	}
	if err != nil {
		return nil, err
	}

	var stored map[string]walletData
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&stored); err != nil {
		return nil, err
	}
	for addr, wd := range stored {
		ws.wallets[addr] = wd.toWallet()
	}
	return ws, nil
}

// Save persists the wallets to file.
func (ws *Wallets) Save(file string) error {
	stored := make(map[string]walletData, len(ws.wallets))
	for addr, w := range ws.wallets {
		stored[addr] = walletData{D: w.PrivateKey.D.Bytes(), PublicKey: w.PublicKey}
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(stored); err != nil {
		return err
	}
	return os.WriteFile(file, buf.Bytes(), 0600)
}

// CreateWallet generates a new wallet, stores it and returns its address.
func (ws *Wallets) CreateWallet() (string, error) {
	w, err := NewWallet()
	if err != nil {
		return "", err
	}
	addr := w.Address()
	ws.wallets[addr] = w
	return addr, nil
}

// GetAddresses returns all wallet addresses, sorted.
func (ws *Wallets) GetAddresses() []string {
	addrs := make([]string, 0, len(ws.wallets))
	for a := range ws.wallets {
		addrs = append(addrs, a)
	}
	sort.Strings(addrs)
	return addrs
}

// GetWallet returns the wallet for an address, if present.
func (ws *Wallets) GetWallet(address string) (*Wallet, bool) {
	w, ok := ws.wallets[address]
	return w, ok
}

// walletData is the gob-friendly on-disk form of a wallet (no curve interface).
type walletData struct {
	D         []byte
	PublicKey []byte
}

func (wd walletData) toWallet() *Wallet {
	var priv ecdsa.PrivateKey
	priv.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(wd.D)
	priv.PublicKey.X = new(big.Int).SetBytes(wd.PublicKey[:fieldSize])
	priv.PublicKey.Y = new(big.Int).SetBytes(wd.PublicKey[fieldSize:])
	return &Wallet{PrivateKey: priv, PublicKey: wd.PublicKey}
}
