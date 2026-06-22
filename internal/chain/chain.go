// Package chain implements a persistent UTXO blockchain stored in BoltDB, with
// ECDSA-signed transactions verified before a block is mined.
package chain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/pow"
	"github.com/thefcan/gochain/internal/tx"
	"github.com/thefcan/gochain/internal/wallet"
)

const (
	blocksBucket = "blocks"
	tipKey       = "l"
)

var (
	ErrChainExists        = errors.New("blockchain already exists")
	ErrNoChain            = errors.New("no blockchain found; create one first")
	ErrInsufficientFunds  = errors.New("not enough funds")
	ErrInvalidTransaction = errors.New("invalid transaction")
)

// Blockchain is a UTXO blockchain backed by a BoltDB database.
type Blockchain struct {
	db  *bbolt.DB
	tip []byte
}

// CreateBlockchain creates a new chain whose genesis block pays the subsidy to
// address. It fails with ErrChainExists if one already exists.
func CreateBlockchain(dbPath, address string) (*Blockchain, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	coinbase, err := tx.NewCoinbaseTX(address, "")
	if err != nil {
		_ = db.Close()
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
		_ = db.Close()
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
		_ = db.Close()
		return nil, err
	}
	return &Blockchain{db: db, tip: tip}, nil
}

// Close releases the database.
func (bc *Blockchain) Close() error { return bc.db.Close() }

// MineBlock verifies every (non-coinbase) transaction, then mines and stores a
// new block as the tip.
func (bc *Blockchain) MineBlock(txs []*tx.Transaction) error {
	for _, t := range txs {
		ok, err := bc.VerifyTransaction(t)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidTransaction
		}
	}

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

func (bc *Blockchain) Iterator() *Iterator {
	return &Iterator{db: bc.db, currentHash: bc.tip}
}

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

// findUTXO returns all unspent outputs locked to pubKeyHash.
func (bc *Blockchain) findUTXO(pubKeyHash []byte) ([]tx.TXOutput, error) {
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
				if out.IsLockedWithKey(pubKeyHash) {
					utxos = append(utxos, out)
				}
			}
			if !t.IsCoinbase() {
				for _, in := range t.Vin {
					if in.UsesKey(pubKeyHash) {
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
	pubKeyHash, err := wallet.PubKeyHashFromAddress(address)
	if err != nil {
		return 0, err
	}
	utxos, err := bc.findUTXO(pubKeyHash)
	if err != nil {
		return 0, err
	}
	balance := 0
	for _, out := range utxos {
		balance += out.Value
	}
	return balance, nil
}

// FindSpendableOutputs collects enough of pubKeyHash's unspent outputs to cover
// amount, returning the total found and a txID->output-indices map.
func (bc *Blockchain) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int, error) {
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
				if out.IsLockedWithKey(pubKeyHash) {
					accumulated += out.Value
					unspent[txID] = append(unspent[txID], outIdx)
					if accumulated >= amount {
						break Work
					}
				}
			}
			if !t.IsCoinbase() {
				for _, in := range t.Vin {
					if in.UsesKey(pubKeyHash) {
						k := hex.EncodeToString(in.Txid)
						spent[k] = append(spent[k], in.Vout)
					}
				}
			}
		}
	}
	return accumulated, unspent, nil
}

// FindTransaction looks up a transaction by ID across the whole chain.
func (bc *Blockchain) FindTransaction(id []byte) (tx.Transaction, error) {
	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			return tx.Transaction{}, err
		}
		if b == nil {
			break
		}
		for _, t := range b.Transactions {
			if bytes.Equal(t.ID, id) {
				return *t, nil
			}
		}
	}
	return tx.Transaction{}, errors.New("transaction not found")
}

func (bc *Blockchain) gatherPrevTXs(t *tx.Transaction) (map[string]tx.Transaction, error) {
	prevTXs := make(map[string]tx.Transaction)
	for _, vin := range t.Vin {
		prevTX, err := bc.FindTransaction(vin.Txid)
		if err != nil {
			return nil, err
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return prevTXs, nil
}

// SignTransaction signs t with priv, resolving the referenced previous outputs.
func (bc *Blockchain) SignTransaction(t *tx.Transaction, priv ecdsa.PrivateKey) error {
	prevTXs, err := bc.gatherPrevTXs(t)
	if err != nil {
		return err
	}
	return t.Sign(priv, prevTXs)
}

// VerifyTransaction verifies t's signatures against the referenced outputs.
func (bc *Blockchain) VerifyTransaction(t *tx.Transaction) (bool, error) {
	if t.IsCoinbase() {
		return true, nil
	}
	prevTXs, err := bc.gatherPrevTXs(t)
	if err != nil {
		return false, err
	}
	return t.Verify(prevTXs)
}

// NewUTXOTransaction builds and signs a transfer of amount from w to address.
func (bc *Blockchain) NewUTXOTransaction(w *wallet.Wallet, to string, amount int) (*tx.Transaction, error) {
	pubKeyHash := wallet.HashPubKey(w.PublicKey)
	acc, validOutputs, err := bc.FindSpendableOutputs(pubKeyHash, amount)
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
			inputs = append(inputs, tx.TXInput{Txid: id, Vout: out, PubKey: w.PublicKey})
		}
	}

	out, err := tx.NewTXOutput(amount, to)
	if err != nil {
		return nil, err
	}
	outputs := []tx.TXOutput{*out}
	if acc > amount {
		change, err := tx.NewTXOutput(acc-amount, w.Address())
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, *change)
	}

	t := &tx.Transaction{Vin: inputs, Vout: outputs}
	if t.ID, err = t.Hash(); err != nil {
		return nil, err
	}
	if err := bc.SignTransaction(t, w.PrivateKey); err != nil {
		return nil, err
	}
	return t, nil
}

// Send builds, signs and mines a transfer of amount from w to address.
func (bc *Blockchain) Send(w *wallet.Wallet, to string, amount int) error {
	t, err := bc.NewUTXOTransaction(w, to, amount)
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

// --- Phase 7: networking support (block replication) ---

var (
	ErrInvalidBlock  = errors.New("invalid block: failed proof of work")
	ErrBlockNotFound = errors.New("block not found")
)

// OpenOrInit opens the chain at dbPath, or returns an empty chain (ready to
// receive blocks from a peer) when none exists yet.
func OpenOrInit(dbPath string) (*Blockchain, error) {
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	var tip []byte
	err = db.View(func(t *bbolt.Tx) error {
		if b := t.Bucket([]byte(blocksBucket)); b != nil {
			tip = append([]byte{}, b.Get([]byte(tipKey))...)
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Blockchain{db: db, tip: tip}, nil
}

// GetBlockHashes returns every block hash from genesis to the tip, in order.
func (bc *Blockchain) GetBlockHashes() ([][]byte, error) {
	var hashes [][]byte
	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			return nil, err
		}
		if b == nil {
			break
		}
		hashes = append(hashes, b.Hash)
	}
	for i, j := 0, len(hashes)-1; i < j; i, j = i+1, j-1 {
		hashes[i], hashes[j] = hashes[j], hashes[i]
	}
	return hashes, nil
}

// GetBlock reads a single block by hash.
func (bc *Blockchain) GetBlock(hash []byte) (*block.Block, error) {
	var b *block.Block
	err := bc.db.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(blocksBucket))
		if bkt == nil {
			return ErrBlockNotFound
		}
		encoded := bkt.Get(hash)
		if encoded == nil {
			return ErrBlockNotFound
		}
		var derr error
		b, derr = block.Deserialize(encoded)
		return derr
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

// HasBlock reports whether the chain already stores a block with this hash.
func (bc *Blockchain) HasBlock(hash []byte) (bool, error) {
	has := false
	err := bc.db.View(func(t *bbolt.Tx) error {
		if bkt := t.Bucket([]byte(blocksBucket)); bkt != nil && bkt.Get(hash) != nil {
			has = true
		}
		return nil
	})
	return has, err
}

// AddReceivedBlock validates a block received from a peer and stores it,
// advancing the tip when the block extends the current chain.
func (bc *Blockchain) AddReceivedBlock(b *block.Block) error {
	if !pow.New(b).Validate() {
		return ErrInvalidBlock
	}
	extendsTip := len(bc.tip) == 0 || bytes.Equal(b.PrevBlockHash, bc.tip)

	err := bc.db.Update(func(t *bbolt.Tx) error {
		bkt, err := t.CreateBucketIfNotExists([]byte(blocksBucket))
		if err != nil {
			return err
		}
		if bkt.Get(b.Hash) != nil {
			return nil // already stored
		}
		ser, err := b.Serialize()
		if err != nil {
			return err
		}
		if err := bkt.Put(b.Hash, ser); err != nil {
			return err
		}
		if extendsTip {
			return bkt.Put([]byte(tipKey), b.Hash)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if extendsTip {
		bc.tip = b.Hash
	}
	return nil
}
