// Package tx defines blockchain transactions using a simplified UTXO model.
// In this phase, ownership is checked against a plain address string; real
// ECDSA signatures are introduced in a later phase.
package tx

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
)

// subsidy is the reward paid by a coinbase transaction.
const subsidy = 10

// TXInput references an output of a previous transaction being spent.
type TXInput struct {
	Txid      []byte // the referenced transaction's ID
	Vout      int    // index of the referenced output
	ScriptSig string // (phase 4) the spender's address
}

// TXOutput holds coins payable to whoever can unlock ScriptPubKey.
type TXOutput struct {
	Value        int
	ScriptPubKey string // (phase 4) the recipient's address
}

// Transaction is a set of inputs spending earlier outputs and new outputs.
type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

// CanUnlockOutputWith reports whether the input was authorised by address.
func (in *TXInput) CanUnlockOutputWith(address string) bool { return in.ScriptSig == address }

// CanBeUnlockedWith reports whether the output is payable to address.
func (out *TXOutput) CanBeUnlockedWith(address string) bool { return out.ScriptPubKey == address }

// IsCoinbase reports whether tx is a coinbase (mining reward) transaction.
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// SetID computes and stores the transaction's SHA-256 hash as its ID.
func (tx *Transaction) SetID() error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(tx); err != nil {
		return err
	}
	hash := sha256.Sum256(buf.Bytes())
	tx.ID = hash[:]
	return nil
}

// NewCoinbaseTX creates a coinbase transaction paying the subsidy to `to`.
func NewCoinbaseTX(to, data string) (*Transaction, error) {
	if data == "" {
		data = fmt.Sprintf("Reward to %s", to)
	}
	tx := &Transaction{
		Vin:  []TXInput{{Txid: []byte{}, Vout: -1, ScriptSig: data}},
		Vout: []TXOutput{{Value: subsidy, ScriptPubKey: to}},
	}
	if err := tx.SetID(); err != nil {
		return nil, err
	}
	return tx, nil
}
