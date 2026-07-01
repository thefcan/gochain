package chain

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/wallet"
)

// TestAddReceivedBlockReorgsToLongerBranch asserts longest-chain fork choice:
// when a competing branch grows longer than the active chain, the node adopts
// it and its balances reflect the new branch.
func TestAddReceivedBlockReorgsToLongerBranch(t *testing.T) {
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
	hashes, _ := a.GetBlockHashes() // [genesis, transfer]
	genesis, _ := a.GetBlock(hashes[0])
	transfer, _ := a.GetBlock(hashes[1])

	// Node B mirrors A: genesis -> transfer (height 2), so owner=6, bob=4.
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()
	if err := b.AddReceivedBlock(genesis); err != nil {
		t.Fatalf("add genesis: %v", err)
	}
	if err := b.AddReceivedBlock(transfer); err != nil {
		t.Fatalf("add transfer: %v", err)
	}
	if bal, _ := b.Balance(owner.Address()); bal != 6 {
		t.Fatalf("before reorg owner balance = %d, want 6", bal)
	}

	// A competing branch built on genesis (not the transfer): two empty blocks,
	// so its height reaches 3 > 2. The transfer never happened on this branch.
	alt1 := block.New(nil, genesis.Hash)
	mineForTest(alt1)
	alt2 := block.New(nil, alt1.Hash)
	mineForTest(alt2)

	// alt1 only ties the current height (2): it is stored but the tip stays put.
	if err := b.AddReceivedBlock(alt1); err != nil {
		t.Fatalf("add alt1: %v", err)
	}
	if bal, _ := b.Balance(bob.Address()); bal != 4 {
		t.Fatalf("after alt1 (no reorg) bob balance = %d, want 4", bal)
	}

	// alt2 makes the branch strictly longer (3): the node reorgs onto it.
	if err := b.AddReceivedBlock(alt2); err != nil {
		t.Fatalf("add alt2: %v", err)
	}

	got, err := b.GetBlockHashes()
	if err != nil {
		t.Fatalf("GetBlockHashes: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("after reorg chain length = %d, want 3", len(got))
	}
	if string(got[len(got)-1]) != string(alt2.Hash) {
		t.Fatal("after reorg the tip is not alt2")
	}
	// On the new branch the transfer never happened: owner keeps the full
	// subsidy and bob has nothing.
	if bal, _ := b.Balance(owner.Address()); bal != 10 {
		t.Errorf("after reorg owner balance = %d, want 10", bal)
	}
	if bal, _ := b.Balance(bob.Address()); bal != 0 {
		t.Errorf("after reorg bob balance = %d, want 0", bal)
	}
}

// TestAddReceivedBlockRejectsOrphan asserts that a block whose parent is unknown
// is rejected rather than stored, since it cannot be validated or ordered.
func TestAddReceivedBlockRejectsOrphan(t *testing.T) {
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()

	orphan := block.New(nil, []byte("unknown-parent"))
	mineForTest(orphan)
	if err := b.AddReceivedBlock(orphan); !errors.Is(err, ErrOrphanBlock) {
		t.Fatalf("orphan: got err = %v, want ErrOrphanBlock", err)
	}
}
