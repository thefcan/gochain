// Package wallet implements ECDSA wallets and Bitcoin-style Base58Check
// addresses (version + RIPEMD160(SHA256(pubkey)) + checksum).
package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"golang.org/x/crypto/ripemd160"
)

const (
	version         = byte(0x00)
	addrChecksumLen = 4
	fieldSize       = 32 // P-256 coordinate width in bytes
)

// Wallet is an ECDSA key pair. PublicKey is the raw X||Y coordinates (64 bytes).
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet generates a fresh ECDSA (P-256) wallet.
func NewWallet() (*Wallet, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Wallet{PrivateKey: *priv, PublicKey: marshalPub(priv.PublicKey.X, priv.PublicKey.Y)}, nil
}

// Address returns the wallet's Base58Check address.
func (w *Wallet) Address() string {
	payload := append([]byte{version}, HashPubKey(w.PublicKey)...)
	full := append(payload, checksum(payload)...)
	return string(Base58Encode(full))
}

// HashPubKey returns RIPEMD160(SHA256(pubKey)) — a 20-byte public key hash.
func HashPubKey(pubKey []byte) []byte {
	sha := sha256.Sum256(pubKey)
	r := ripemd160.New()
	r.Write(sha[:])
	return r.Sum(nil)
}

// ValidateAddress reports whether address is well-formed (valid checksum).
func ValidateAddress(address string) bool {
	full := Base58Decode([]byte(address))
	if len(full) < addrChecksumLen+1 {
		return false
	}
	actual := full[len(full)-addrChecksumLen:]
	payload := full[:len(full)-addrChecksumLen]
	return bytes.Equal(actual, checksum(payload))
}

// checksum returns the first bytes of the double SHA-256 of payload.
func checksum(payload []byte) []byte {
	first := sha256.Sum256(payload)
	second := sha256.Sum256(first[:])
	return second[:addrChecksumLen]
}

// marshalPub encodes X and Y as fixed-width 32-byte big-endian halves.
func marshalPub(x, y *big.Int) []byte {
	pub := make([]byte, 2*fieldSize)
	x.FillBytes(pub[:fieldSize])
	y.FillBytes(pub[fieldSize:])
	return pub
}
