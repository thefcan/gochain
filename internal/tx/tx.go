// Package tx defines blockchain transactions using a UTXO model secured by
// ECDSA signatures. Outputs are locked to a public-key hash; spending an output
// requires a signature from the matching private key.
package tx

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/thefcan/gochain/internal/wallet"
)

const (
	subsidy = 10 // mining reward
	sigLen  = 64 // ECDSA P-256 signature: r||s, 32 bytes each
	keyLen  = 65 // uncompressed SEC1 public key: 0x04 || X || Y
)

// Errors returned when a transaction references outputs that cannot be resolved.
var (
	ErrPrevTxNotFound     = errors.New("previous transaction not found")
	ErrInvalidOutputIndex = errors.New("input references a non-existent output index")
)

// TXInput spends an output of a previous transaction.
type TXInput struct {
	Txid      []byte // referenced transaction ID
	Vout      int    // referenced output index
	Signature []byte // r||s over the spend
	PubKey    []byte // spender's uncompressed public key
}

// TXOutput holds coins locked to a public-key hash.
type TXOutput struct {
	Value      int
	PubKeyHash []byte
}

// Transaction is a set of inputs spending earlier outputs and new outputs.
type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

// UsesKey reports whether the input is signed by the owner of pubKeyHash.
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	return bytes.Equal(wallet.HashPubKey(in.PubKey), pubKeyHash)
}

// IsLockedWithKey reports whether the output is payable to pubKeyHash.
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}

// NewTXOutput creates an output of value locked to address.
func NewTXOutput(value int, address string) (*TXOutput, error) {
	pubKeyHash, err := wallet.PubKeyHashFromAddress(address)
	if err != nil {
		return nil, err
	}
	return &TXOutput{Value: value, PubKeyHash: pubKeyHash}, nil
}

// IsCoinbase reports whether tx is a coinbase (mining reward) transaction.
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Hash returns the SHA-256 hash of the transaction with its ID field cleared.
func (tx *Transaction) Hash() ([]byte, error) {
	clone := *tx
	clone.ID = nil
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(clone); err != nil {
		return nil, err
	}
	h := sha256.Sum256(buf.Bytes())
	return h[:], nil
}

// NewCoinbaseTX creates a coinbase transaction paying the subsidy to `to`.
func NewCoinbaseTX(to, data string) (*Transaction, error) {
	out, err := NewTXOutput(subsidy, to)
	if err != nil {
		return nil, err
	}
	tx := &Transaction{
		Vin:  []TXInput{{Txid: []byte{}, Vout: -1, PubKey: []byte(data)}},
		Vout: []TXOutput{*out},
	}
	id, err := tx.Hash()
	if err != nil {
		return nil, err
	}
	tx.ID = id
	return tx, nil
}

// TrimmedCopy returns a copy with input signatures and pubkeys cleared, used as
// the basis for signing and verifying.
func (tx *Transaction) TrimmedCopy() Transaction {
	inputs := make([]TXInput, 0, len(tx.Vin))
	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{Txid: vin.Txid, Vout: vin.Vout})
	}
	outputs := make([]TXOutput, 0, len(tx.Vout))
	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{Value: vout.Value, PubKeyHash: vout.PubKeyHash})
	}
	return Transaction{ID: tx.ID, Vin: inputs, Vout: outputs}
}

// Sign signs each input with priv, using the referenced previous transactions.
func (tx *Transaction) Sign(priv ecdsa.PrivateKey, prevTXs map[string]Transaction) error {
	if tx.IsCoinbase() {
		return nil
	}
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			return ErrPrevTxNotFound
		}
	}

	txCopy := tx.TrimmedCopy()
	for inID, vin := range txCopy.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		if vin.Vout < 0 || vin.Vout >= len(prevTX.Vout) {
			return ErrInvalidOutputIndex
		}
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubKeyHash

		dataToSign, err := txCopy.Hash()
		if err != nil {
			return err
		}
		txCopy.Vin[inID].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &priv, dataToSign)
		if err != nil {
			return err
		}
		sig := make([]byte, sigLen)
		r.FillBytes(sig[:sigLen/2])
		s.FillBytes(sig[sigLen/2:])
		tx.Vin[inID].Signature = sig
	}
	return nil
}

// Verify checks every input signature against the referenced previous outputs.
func (tx *Transaction) Verify(prevTXs map[string]Transaction) (bool, error) {
	if tx.IsCoinbase() {
		return true, nil
	}
	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			return false, ErrPrevTxNotFound
		}
	}

	txCopy := tx.TrimmedCopy()
	for inID, vin := range tx.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		if vin.Vout < 0 || vin.Vout >= len(prevTX.Vout) {
			return false, nil
		}
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubKeyHash

		dataToVerify, err := txCopy.Hash()
		if err != nil {
			return false, err
		}
		txCopy.Vin[inID].PubKey = nil

		if len(vin.Signature) != sigLen || len(vin.PubKey) != keyLen {
			return false, nil
		}
		r := new(big.Int).SetBytes(vin.Signature[:sigLen/2])
		s := new(big.Int).SetBytes(vin.Signature[sigLen/2:])
		pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), vin.PubKey)
		if err != nil {
			return false, nil
		}
		if !ecdsa.Verify(pub, dataToVerify, r, s) {
			return false, nil
		}
	}
	return true, nil
}
