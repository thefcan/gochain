package chain

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/pow"
	"github.com/thefcan/gochain/internal/wallet"
)

// TestAddReceivedBlockRejectsForgedTx asserts that a peer cannot smuggle an
// invalid transaction past a node just because the block carries a valid proof
// of work. After tampering with a signed output and re-mining (so the PoW is
// valid again), the block must be rejected on transaction verification.
// Before AddReceivedBlock verified transactions, this forged block was accepted.
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
	nonce, hash := pow.New(forged).Run()
	forged.Nonce = nonce
	forged.Hash = hash

	if err := b.AddReceivedBlock(forged); !errors.Is(err, ErrInvalidTransaction) {
		t.Fatalf("forged block: got err = %v, want ErrInvalidTransaction", err)
	}
}
