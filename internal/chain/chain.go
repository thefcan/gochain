// Package chain implements a persistent blockchain stored in a BoltDB file:
// blocks live in a bucket keyed by their hash, with a special key tracking the
// tip. Iteration walks the chain from the tip back to genesis.
package chain

import (
	"fmt"

	"go.etcd.io/bbolt"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/pow"
)

const (
	blocksBucket = "blocks"
	tipKey       = "l"
)

// Blockchain is a blockchain backed by a BoltDB database.
type Blockchain struct {
	db  *bbolt.DB
	tip []byte
}

// Open opens (or creates) the blockchain stored at dbPath. If the database has
// no chain yet, a genesis block is mined and stored.
func Open(dbPath string) (*Blockchain, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	var tip []byte
	err = db.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(blocksBucket))
		if bkt != nil {
			tip = append([]byte{}, bkt.Get([]byte(tipKey))...)
			return nil
		}

		// Fresh database: mine and store the genesis block.
		genesis := block.NewGenesis()
		mine(genesis)
		bkt, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			return err
		}
		ser, err := genesis.Serialize()
		if err != nil {
			return err
		}
		if err := bkt.Put(genesis.Hash, ser); err != nil {
			return err
		}
		if err := bkt.Put([]byte(tipKey), genesis.Hash); err != nil {
			return err
		}
		tip = genesis.Hash
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Blockchain{db: db, tip: tip}, nil
}

// Close releases the database.
func (bc *Blockchain) Close() error { return bc.db.Close() }

// AddBlock mines a new block carrying data and persists it as the new tip.
func (bc *Blockchain) AddBlock(data string) error {
	newBlock := block.New(data, bc.tip)
	mine(newBlock)

	err := bc.db.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte(blocksBucket))
		ser, err := newBlock.Serialize()
		if err != nil {
			return err
		}
		if err := bkt.Put(newBlock.Hash, ser); err != nil {
			return err
		}
		return bkt.Put([]byte(tipKey), newBlock.Hash)
	})
	if err != nil {
		return err
	}
	bc.tip = newBlock.Hash
	return nil
}

// mine runs proof of work on b and records the winning nonce and hash.
func mine(b *block.Block) {
	nonce, hash := pow.New(b).Run()
	b.Nonce = nonce
	b.Hash = hash
}

// Iterator walks the chain from the tip back to the genesis block.
type Iterator struct {
	db          *bbolt.DB
	currentHash []byte
}

// Iterator returns a new iterator positioned at the tip.
func (bc *Blockchain) Iterator() *Iterator {
	return &Iterator{db: bc.db, currentHash: bc.tip}
}

// Next returns the next block (tip first), or (nil, nil) when the chain is
// exhausted.
func (it *Iterator) Next() (*block.Block, error) {
	if len(it.currentHash) == 0 {
		return nil, nil
	}

	var b *block.Block
	err := it.db.View(func(tx *bbolt.Tx) error {
		encoded := tx.Bucket([]byte(blocksBucket)).Get(it.currentHash)
		if encoded == nil {
			return fmt.Errorf("block %x not found", it.currentHash)
		}
		var derr error
		b, derr = block.Deserialize(encoded)
		return derr
	})
	if err != nil {
		return nil, err
	}
	it.currentHash = b.PrevBlockHash
	return b, nil
}
