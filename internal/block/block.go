// Package block defines the Block type — a single, hash-linked entry in the
// chain — and how its hash is computed.
package block

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"time"
)

// Block is one link in the chain. Each block commits to the previous block's
// hash, forming a tamper-evident sequence.
type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
}

// New creates a block from data and the previous block's hash, computing its own
// hash from its contents.
func New(data string, prevBlockHash []byte) *Block {
	b := &Block{
		Timestamp:     time.Now().Unix(),
		Data:          []byte(data),
		PrevBlockHash: prevBlockHash,
	}
	b.Hash = b.computeHash()
	return b
}

// NewGenesis creates the first block of a chain, which has no predecessor.
func NewGenesis() *Block {
	return New("Genesis Block", []byte{})
}

// computeHash returns the SHA-256 digest of the block's headers.
func (b *Block) computeHash() []byte {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})
	sum := sha256.Sum256(headers)
	return sum[:]
}
