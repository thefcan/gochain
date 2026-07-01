package chain_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/chain"
	"github.com/thefcan/gochain/internal/wallet"
)

// TestValueConservationAndNoOverspend exercises the real end-to-end path
// (create → send → verify → mine, persisted in BoltDB) and asserts two
// invariants:
//
//  1. Value is conserved across a transfer (sender + recipient balances still
//     sum to the genesis subsidy).
//  2. A wallet cannot spend more than its confirmed balance — the protocol
//     rejects the overspend (ErrInsufficientFunds) and leaves balances intact,
//     so a spent output cannot be double-spent through the normal API.
//
// This is an integration test: it mines real blocks (proof of work), so it is
// slower than the pure property tests.
func TestValueConservationAndNoOverspend(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "chain.db")

	alice, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet (alice): %v", err)
	}
	bob, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet (bob): %v", err)
	}

	bc, err := chain.CreateBlockchain(dbPath, alice.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	defer func() { _ = bc.Close() }()

	// Genesis pays the subsidy (10) to Alice.
	const subsidy = 10
	if bal, err := bc.Balance(alice.Address()); err != nil {
		t.Fatalf("Balance (alice, genesis): %v", err)
	} else if bal != subsidy {
		t.Fatalf("genesis balance = %d, want %d", bal, subsidy)
	}

	// Alice sends 6 to Bob.
	if err := bc.Send(alice, bob.Address(), 6); err != nil {
		t.Fatalf("Send(alice -> bob, 6): %v", err)
	}

	balA, err := bc.Balance(alice.Address())
	if err != nil {
		t.Fatalf("Balance (alice): %v", err)
	}
	balB, err := bc.Balance(bob.Address())
	if err != nil {
		t.Fatalf("Balance (bob): %v", err)
	}
	if balA != 4 {
		t.Fatalf("alice balance after send = %d, want 4", balA)
	}
	if balB != 6 {
		t.Fatalf("bob balance after send = %d, want 6", balB)
	}
	if balA+balB != subsidy {
		t.Fatalf("value not conserved: %d + %d = %d, want %d", balA, balB, balA+balB, subsidy)
	}

	// Alice now has only 4, but tries to send 6 again: must be rejected.
	err = bc.Send(alice, bob.Address(), 6)
	if !errors.Is(err, chain.ErrInsufficientFunds) {
		t.Fatalf("overspend: got err = %v, want ErrInsufficientFunds", err)
	}

	// The failed overspend must not have changed any balances.
	balA2, _ := bc.Balance(alice.Address())
	balB2, _ := bc.Balance(bob.Address())
	if balA2 != 4 || balB2 != 6 {
		t.Fatalf("balances changed after a rejected send: alice=%d bob=%d", balA2, balB2)
	}
}
