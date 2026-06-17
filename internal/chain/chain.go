// Package chain implements an in-memory blockchain: an ordered, hash-linked list
// of blocks starting from a genesis block.
package chain

import "github.com/thefcan/gochain/internal/block"

// Blockchain is an ordered chain of blocks held in memory.
type Blockchain struct {
	blocks []*block.Block
}

// New returns a blockchain containing only the genesis block.
func New() *Blockchain {
	return &Blockchain{blocks: []*block.Block{block.NewGenesis()}}
}

// AddBlock appends a new block carrying data, linked to the current tip.
func (bc *Blockchain) AddBlock(data string) {
	tip := bc.blocks[len(bc.blocks)-1]
	bc.blocks = append(bc.blocks, block.New(data, tip.Hash))
}

// Blocks returns the chain's blocks in order, oldest first.
func (bc *Blockchain) Blocks() []*block.Block {
	return bc.blocks
}
