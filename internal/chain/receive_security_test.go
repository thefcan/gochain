package chain

import (
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/pow"
	"github.com/thefcan/gochain/internal/tx"
	"github.com/thefcan/gochain/internal/wallet"
)

// mine gives a block a valid proof of work in place.
func mineForTest(b *block.Block) {
	nonce, hash := pow.New(b).Run()
	b.Nonce = nonce
	b.Hash = hash
}

// TestAddReceivedBlockRejectsForgedTx asserts that a peer cannot smuggle an
// invalid transaction past a node just because the block carries a valid proof
// of work. After tampering with a signed output and re-mining (so the PoW is
// valid again), the block must be rejected on transaction verification.
func TestAddReceivedBlockRejectsForgedTx(t *testing.T) {
	owner, _ := wallet.NewWallet()
	a, err := CreateBlockchain(filepath.Join(t.TempDir(), "a.db"), owner.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	defer a.Close()

	bob, _ := wallet.NewWallet()
	if err := a.Send(owner, bob.Address(), 4); err != nil {
		t.Fatalf("Send: %v", err)
	}

	hashes, err := a.GetBlockHashes()
	if err != nil {
		t.Fatalf("GetBlockHashes: %v", err)
	}

	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()

	// Genesis replicates fine (coinbase needs no signature check).
	genesis, err := a.GetBlock(hashes[0])
	if err != nil {
		t.Fatalf("GetBlock(genesis): %v", err)
	}
	if err := b.AddReceivedBlock(genesis); err != nil {
		t.Fatalf("AddReceivedBlock(genesis): %v", err)
	}

	// Tamper with a signed output in the transfer block, then re-mine so the
	// proof of work is valid again — only the signature is now wrong.
	forged, err := a.GetBlock(hashes[1])
	if err != nil {
		t.Fatalf("GetBlock(transfer): %v", err)
	}
	forged.Transactions[0].Vout[0].Value += 1000 // steal coins; invalidates the signature
	mineForTest(forged)

	if err := b.AddReceivedBlock(forged); !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("forged block: got err = %v, want ErrInvalidTransaction", err)
	}
}

// TestAddReceivedBlockRejectsBadCoinbaseValue asserts that a "genesis" block
// whose coinbase over-pays itself is rejected, so a peer cannot mint coins.
func TestAddReceivedBlockRejectsBadCoinbaseValue(t *testing.T) {
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()

	victim, _ := wallet.NewWallet()
	badCoinbase := &tx.Transaction{
		Vin:  []tx.TXInput{{Txid: []byte{}, Vout: -1, PubKey: []byte("x")}},
		Vout: []tx.TXOutput{{Value: 1000, PubKeyHash: wallet.HashPubKey(victim.PublicKey)}},
	}
	if badCoinbase.ID, err = badCoinbase.Hash(); err != nil {
		t.Fatalf("Hash: %v", err)
	}
	blk := block.New([]*tx.Transaction{badCoinbase}, []byte{})
	mineForTest(blk)

	if err := b.AddReceivedBlock(blk); !errors.Is(err, ErrInvalidCoinbase) {
		t.Fatalf("over-paying coinbase: got err = %v, want ErrInvalidCoinbase", err)
	}
}

// TestAddReceivedBlockRejectsCoinbaseAfterGenesis asserts that a block carrying
// a coinbase after the chain already exists is rejected — coinbases (new coins)
// are only legal in the genesis block.
func TestAddReceivedBlockRejectsCoinbaseAfterGenesis(t *testing.T) {
	owner, _ := wallet.NewWallet()
	a, err := CreateBlockchain(filepath.Join(t.TempDir(), "a.db"), owner.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	defer a.Close()

	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()

	hashes, _ := a.GetBlockHashes()
	genesis, _ := a.GetBlock(hashes[0])
	if err := b.AddReceivedBlock(genesis); err != nil {
		t.Fatalf("AddReceivedBlock(genesis): %v", err)
	}

	// A well-formed coinbase (correct value) but in a non-genesis block.
	cb, err := tx.NewCoinbaseTX(owner.Address(), "reward")
	if err != nil {
		t.Fatalf("NewCoinbaseTX: %v", err)
	}
	blk := block.New([]*tx.Transaction{cb}, b.tip)
	mineForTest(blk)

	if err := b.AddReceivedBlock(blk); !errors.Is(err, ErrInvalidCoinbase) {
		t.Fatalf("coinbase after genesis: got err = %v, want ErrInvalidCoinbase", err)
	}
}

// TestAddReceivedBlockRejectsDoubleSpend asserts that a validly-signed block
// that spends an output already spent by the chain is rejected.
func TestAddReceivedBlockRejectsDoubleSpend(t *testing.T) {
	owner, _ := wallet.NewWallet()
	a, err := CreateBlockchain(filepath.Join(t.TempDir(), "a.db"), owner.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	defer a.Close()

	bob, _ := wallet.NewWallet()
	if err := a.Send(owner, bob.Address(), 4); err != nil {
		t.Fatalf("Send: %v", err)
	}
	hashes, err := a.GetBlockHashes()
	if err != nil {
		t.Fatalf("GetBlockHashes: %v", err)
	}

	// Replicate the full chain (genesis + transfer) into B, so the genesis
	// coinbase output is now spent by the transfer block.
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()
	for _, h := range hashes {
		blk, err := a.GetBlock(h)
		if err != nil {
			t.Fatalf("GetBlock: %v", err)
		}
		if err := b.AddReceivedBlock(blk); err != nil {
			t.Fatalf("AddReceivedBlock: %v", err)
		}
	}

	// Craft a validly-signed transaction that spends the genesis coinbase
	// output a SECOND time.
	genesis, _ := a.GetBlock(hashes[0])
	coinbase := genesis.Transactions[0]
	ds := &tx.Transaction{
		Vin:  []tx.TXInput{{Txid: coinbase.ID, Vout: 0, PubKey: owner.PublicKey}},
		Vout: []tx.TXOutput{{Value: tx.Subsidy, PubKeyHash: wallet.HashPubKey(bob.PublicKey)}},
	}
	if ds.ID, err = ds.Hash(); err != nil {
		t.Fatalf("ds.Hash: %v", err)
	}
	prevTXs := map[string]tx.Transaction{hex.EncodeToString(coinbase.ID): *coinbase}
	if err := ds.Sign(owner.PrivateKey, prevTXs); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	dsBlock := block.New([]*tx.Transaction{ds}, b.tip)
	mineForTest(dsBlock)

	if err := b.AddReceivedBlock(dsBlock); !errors.Is(err, ErrDoubleSpend) {
		t.Fatalf("double-spend: got err = %v, want ErrDoubleSpend", err)
	}
}
