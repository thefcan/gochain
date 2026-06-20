// Package block defines the Block type — a single, hash-linked entry in the
// chain — and its gob (de)serialization. A block carries a set of transactions;
// its hash and nonce are produced by proof of work (see the pow package).
package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"time"

	"github.com/thefcan/gochain/internal/tx"
)

// Block is one link in the chain.
type Block struct {
	Timestamp     int64
	Transactions  []*tx.Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

// New creates an unmined block from transactions and the previous block's hash.
func New(transactions []*tx.Transaction, prevBlockHash []byte) *Block {
	return &Block{
		Timestamp:     time.Now().Unix(),
		Transactions:  transactions,
		PrevBlockHash: prevBlockHash,
	}
}

// HashTransactions returns a single hash over the block's transaction IDs (a
// simplified Merkle root) for use in proof of work.
func (b *Block) HashTransactions() []byte {
	var ids [][]byte
	for _, t := range b.Transactions {
		ids = append(ids, t.ID)
	}
	sum := sha256.Sum256(bytes.Join(ids, []byte{}))
	return sum[:]
}

// Serialize encodes the block to bytes for storage.
func (b *Block) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a block previously produced by Serialize.
func Deserialize(data []byte) (*Block, error) {
	var b Block
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}
