// Package block defines the Block type — a single, hash-linked entry in the
// chain. The block's hash and nonce are produced by proof of work (see the pow
// package); this package keeps the block as plain data.
package block

import "time"

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
