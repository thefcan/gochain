// Package block defines the Block type — a single, hash-linked entry in the
// chain — and its gob (de)serialization for storage. The block's hash and nonce
// are produced by proof of work (see the pow package).
package block

import (
	"bytes"
	"encoding/gob"
	"time"
)

// Block is one link in the chain. Each block commits to the previous block's
// hash, forming a tamper-evident sequence. Hash and Nonce are filled in by
// proof of work after construction.
type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

// New creates an unmined block from data and the previous block's hash.
func New(data string, prevBlockHash []byte) *Block {
	return &Block{
		Timestamp:     time.Now().Unix(),
		Data:          []byte(data),
		PrevBlockHash: prevBlockHash,
	}
}

// NewGenesis creates the (still unmined) first block of a chain.
func NewGenesis() *Block {
	return New("Genesis Block", []byte{})
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
