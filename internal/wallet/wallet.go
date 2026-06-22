// Package wallet implements ECDSA wallets and Bitcoin-style Base58Check
// addresses (version + RIPEMD160(SHA256(pubkey)) + checksum).
package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"

	//nolint:staticcheck // RIPEMD-160 is part of the Bitcoin address spec; required, not a free choice.
	"golang.org/x/crypto/ripemd160"
)

const (
	version         = byte(0x00)
	addrChecksumLen = 4
)

// Wallet is an ECDSA (P-256) key pair. PublicKey is the uncompressed SEC1
// encoding (0x04 || X || Y).
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet generates a fresh ECDSA wallet.
func NewWallet() (*Wallet, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub, err := priv.PublicKey.Bytes()
	if err != nil {
		return nil, err
	}
	return &Wallet{PrivateKey: *priv, PublicKey: pub}, nil
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
