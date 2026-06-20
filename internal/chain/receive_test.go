package chain

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/block"
	"github.com/thefcan/gochain/internal/wallet"
)

func TestAddReceivedBlockReplicatesChain(t *testing.T) {
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

	hashesA, err := a.GetBlockHashes()
	if err != nil {
		t.Fatalf("GetBlockHashes: %v", err)
	}

	// Replicate A's blocks into an empty chain B (as a peer sync would).
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()
	for _, h := range hashesA {
		blk, err := a.GetBlock(h)
		if err != nil {
			t.Fatalf("GetBlock: %v", err)
		}
		if err := b.AddReceivedBlock(blk); err != nil {
			t.Fatalf("AddReceivedBlock: %v", err)
		}
	}

	hashesB, _ := b.GetBlockHashes()
	if len(hashesB) != len(hashesA) {
		t.Fatalf("B has %d blocks, want %d", len(hashesB), len(hashesA))
	}
	for i := range hashesA {
		if string(hashesA[i]) != string(hashesB[i]) {
			t.Errorf("block %d hash mismatch", i)
		}
	}
	if balB, _ := b.Balance(owner.Address()); balB != 6 {
		t.Errorf("replicated owner balance = %d, want 6", balB)
	}
}

func TestAddReceivedBlockRejectsInvalidPoW(t *testing.T) {
	b, err := OpenOrInit(filepath.Join(t.TempDir(), "b.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer b.Close()

	bad := block.New(nil, []byte{}) // never mined -> invalid PoW
	bad.Hash = []byte("fake")
	if err := b.AddReceivedBlock(bad); !errors.Is(err, ErrInvalidBlock) {
		t.Errorf("err = %v, want ErrInvalidBlock", err)
	}
}
