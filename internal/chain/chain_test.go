package chain

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/pow"
)

func tempChain(t *testing.T, address string) *Blockchain {
	t.Helper()
	bc, err := CreateBlockchain(filepath.Join(t.TempDir(), "test.db"), address)
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	t.Cleanup(func() { bc.Close() })
	return bc
}

func TestCreateBlockchainGivesGenesisReward(t *testing.T) {
	bc := tempChain(t, "alice")
	bal, err := bc.Balance("alice")
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if bal != 10 {
		t.Errorf("alice balance = %d, want 10 (subsidy)", bal)
	}
}

func TestSendTransfersFundsWithChange(t *testing.T) {
	bc := tempChain(t, "alice")
	if err := bc.Send("alice", "bob", 4); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if a, _ := bc.Balance("alice"); a != 6 {
		t.Errorf("alice balance = %d, want 6 (change)", a)
	}
	if b, _ := bc.Balance("bob"); b != 4 {
		t.Errorf("bob balance = %d, want 4", b)
	}
}

func TestSendInsufficientFunds(t *testing.T) {
	bc := tempChain(t, "alice")
	if err := bc.Send("alice", "bob", 999); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("Send too much: err = %v, want ErrInsufficientFunds", err)
	}
}

func TestOpenErrorsWhenNoChain(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "none.db"))
	if !errors.Is(err, ErrNoChain) {
		t.Errorf("Open on empty: err = %v, want ErrNoChain", err)
	}
}

func TestBalancePersistsAndBlocksValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	bc, err := CreateBlockchain(path, "alice")
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	if err := bc.Send("alice", "bob", 3); err != nil {
		t.Fatalf("Send: %v", err)
	}
	bc.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer reopened.Close()

	if b, _ := reopened.Balance("bob"); b != 3 {
		t.Errorf("bob balance after reopen = %d, want 3", b)
	}
	// Every block must still pass proof of work.
	it := reopened.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if b == nil {
			break
		}
		if !pow.New(b).Validate() {
			t.Errorf("block %x failed PoW validation", b.Hash)
		}
	}
}
