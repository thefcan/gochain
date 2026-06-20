// Package chain implements a persistent UTXO blockchain stored in BoltDB.
// It mines blocks, tracks unspent transaction outputs and builds transfers.
package chain

import (
	"encoding/hex"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/pow"
	"github.com/thefcan/gochain/internal/tx"
)

const (
	blocksBucket = "blocks"
	tipKey       = "l"
)

// Sentinel errors.
var (
	ErrChainExists       = errors.New("blockchain already exists")
	ErrNoChain           = errors.New("no blockchain found; create one first")
	ErrInsufficientFunds = errors.New("not enough funds")
)

// Blockchain is a UTXO blockchain backed by a BoltDB database.
type Blockchain struct {
	db  *bbolt.DB
	tip []byte
}

// CreateBlockchain creates a new chain whose genesis block pays the mining
// subsidy to address. It fails with ErrChainExists if one already exists.
func CreateBlockchain(dbPath, address string) (*Blockchain, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	coinbase, err := tx.NewCoinbaseTX(address, "")
	if err != nil {
		db.Close()
		return nil, err
	}
	genesis := block.New([]*tx.Transaction{coinbase}, []byte{})
	mine(genesis)

	err = db.Update(func(t *bbolt.Tx) error {
		if t.Bucket([]byte(blocksBucket)) != nil {
			return ErrChainExists
		}
		bkt, err := t.CreateBucket([]byte(blocksBucket))
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
		return bkt.Put([]byte(tipKey), genesis.Hash)
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Blockchain{db: db, tip: genesis.Hash}, nil
}

// Open opens an existing chain, failing with ErrNoChain if none exists.
func Open(dbPath string) (*Blockchain, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	var tip []byte
	err = db.View(func(t *bbolt.Tx) error {
		b := t.Bucket([]byte(blocksBucket))
		if b == nil {
			return ErrNoChain
		}
		tip = append([]byte{}, b.Get([]byte(tipKey))...)
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

// MineBlock mines a new block containing txs and persists it as the new tip.
func (bc *Blockchain) MineBlock(txs []*tx.Transaction) error {
	newBlock := block.New(txs, bc.tip)
	mine(newBlock)

	err := bc.db.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(blocksBucket))
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

func mine(b *block.Block) {
	nonce, hash := pow.New(b).Run()
	b.Nonce = nonce
	b.Hash = hash
}

// Iterator walks the chain from the tip back to genesis.
type Iterator struct {
	db          *bbolt.DB
	currentHash []byte
}

// Iterator returns a new iterator positioned at the tip.
func (bc *Blockchain) Iterator() *Iterator {
	return &Iterator{db: bc.db, currentHash: bc.tip}
}

// Next returns the next block (tip first), or (nil, nil) at the end.
func (it *Iterator) Next() (*block.Block, error) {
	if len(it.currentHash) == 0 {
		return nil, nil
	}
	var b *block.Block
	err := it.db.View(func(t *bbolt.Tx) error {
		encoded := t.Bucket([]byte(blocksBucket)).Get(it.currentHash)
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

// FindUTXO returns all unspent outputs payable to address. Iterating tip->genesis
// means a spending input is always seen before the output it spends, so spent
// outputs are correctly excluded (no double counting).
func (bc *Blockchain) FindUTXO(address string) ([]tx.TXOutput, error) {
	var utxos []tx.TXOutput
	spent := make(map[string][]int)

	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			return nil, err
		}
		if b == nil {
			break
		}
		for _, t := range b.Transactions {
			txID := hex.EncodeToString(t.ID)
			for outIdx, out := range t.Vout {
				if isSpent(spent, txID, outIdx) {
					continue
				}
				if out.CanBeUnlockedWith(address) {
					utxos = append(utxos, out)
				}
			}
			if !t.IsCoinbase() {
				for _, in := range t.Vin {
					if in.CanUnlockOutputWith(address) {
						k := hex.EncodeToString(in.Txid)
						spent[k] = append(spent[k], in.Vout)
					}
				}
			}
		}
	}
	return utxos, nil
}

// Balance returns the total unspent value held by address.
func (bc *Blockchain) Balance(address string) (int, error) {
	utxos, err := bc.FindUTXO(address)
	if err != nil {
		return 0, err
	}
	balance := 0
	for _, out := range utxos {
		balance += out.Value
	}
	return balance, nil
}

// FindSpendableOutputs collects enough of address's unspent outputs to cover
// amount, returning the total found and a txID->output-indices map.
func (bc *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int, error) {
	unspent := make(map[string][]int)
	spent := make(map[string][]int)
	accumulated := 0

	it := bc.Iterator()
Work:
	for {
		b, err := it.Next()
		if err != nil {
			return 0, nil, err
		}
		if b == nil {
			break
		}
		for _, t := range b.Transactions {
			txID := hex.EncodeToString(t.ID)
			for outIdx, out := range t.Vout {
				if isSpent(spent, txID, outIdx) {
					continue
				}
				if out.CanBeUnlockedWith(address) {
					accumulated += out.Value
					unspent[txID] = append(unspent[txID], outIdx)
					if accumulated >= amount {
						break Work
					}
				}
			}
			if !t.IsCoinbase() {
				for _, in := range t.Vin {
					if in.CanUnlockOutputWith(address) {
						k := hex.EncodeToString(in.Txid)
						spent[k] = append(spent[k], in.Vout)
					}
				}
			}
		}
	}
	return accumulated, unspent, nil
}

// NewUTXOTransaction builds (but does not mine) a transfer of amount from -> to.
func (bc *Blockchain) NewUTXOTransaction(from, to string, amount int) (*tx.Transaction, error) {
	acc, validOutputs, err := bc.FindSpendableOutputs(from, amount)
	if err != nil {
		return nil, err
	}
	if acc < amount {
		return nil, ErrInsufficientFunds
	}

	var inputs []tx.TXInput
	for txid, outs := range validOutputs {
		id, err := hex.DecodeString(txid)
		if err != nil {
			return nil, err
		}
		for _, out := range outs {
			inputs = append(inputs, tx.TXInput{Txid: id, Vout: out, ScriptSig: from})
		}
	}

	outputs := []tx.TXOutput{{Value: amount, ScriptPubKey: to}}
	if acc > amount {
		outputs = append(outputs, tx.TXOutput{Value: acc - amount, ScriptPubKey: from}) // change
	}

	t := &tx.Transaction{Vin: inputs, Vout: outputs}
	if err := t.SetID(); err != nil {
		return nil, err
	}
	return t, nil
}

// Send mines a block transferring amount from -> to.
func (bc *Blockchain) Send(from, to string, amount int) error {
	t, err := bc.NewUTXOTransaction(from, to, amount)
	if err != nil {
		return err
	}
	return bc.MineBlock([]*tx.Transaction{t})
}

func isSpent(spent map[string][]int, txID string, outIdx int) bool {
	for _, i := range spent[txID] {
		if i == outIdx {
			return true
		}
	}
	return false
}
