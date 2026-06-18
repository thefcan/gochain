// Package chain implements an in-memory blockchain: an ordered, hash-linked list
// of blocks, each sealed with proof of work, starting from a genesis block.
package chain

import (
	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/pow"
)

// Blockchain is an ordered chain of mined blocks held in memory.
type Blockchain struct {
	blocks []*block.Block
}

// New returns a blockchain containing only the (mined) genesis block.
func New() *Blockchain {
	genesis := block.NewGenesis()
	mine(genesis)
	return &Blockchain{blocks: []*block.Block{genesis}}
}

// AddBlock mines a new block carrying data and appends it to the chain.
func (bc *Blockchain) AddBlock(data string) {
	tip := bc.blocks[len(bc.blocks)-1]
	b := block.New(data, tip.Hash)
	mine(b)
	bc.blocks = append(bc.blocks, b)
}

// Blocks returns the chain's blocks in order, oldest first.
func (bc *Blockchain) Blocks() []*block.Block {
	return bc.blocks
}

// mine runs proof of work on b and records the winning nonce and hash.
func mine(b *block.Block) {
	nonce, hash := pow.New(b).Run()
	b.Nonce = nonce
	b.Hash = hash
}
