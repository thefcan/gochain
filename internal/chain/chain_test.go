package chain

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/tx"
	"github.com/thefcan/gochain/internal/wallet"
)

func setup(t *testing.T) (*Blockchain, *wallet.Wallet) {
	t.Helper()
	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet: %v", err)
	}
	bc, err := CreateBlockchain(filepath.Join(t.TempDir(), "test.db"), w.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	t.Cleanup(func() { bc.Close() })
	return bc, w
}

func TestGenesisRewardAndSignedTransfer(t *testing.T) {
	bc, alice := setup(t)
	if bal, _ := bc.Balance(alice.Address()); bal != 10 {
		t.Fatalf("alice balance = %d, want 10", bal)
	}

	bob, _ := wallet.NewWallet()
	if err := bc.Send(alice, bob.Address(), 4); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if a, _ := bc.Balance(alice.Address()); a != 6 {
		t.Errorf("alice balance = %d, want 6 (change)", a)
	}
	if b, _ := bc.Balance(bob.Address()); b != 4 {
		t.Errorf("bob balance = %d, want 4", b)
	}
}

func TestEveryTransactionVerifies(t *testing.T) {
	bc, alice := setup(t)
	bob, _ := wallet.NewWallet()
	if err := bc.Send(alice, bob.Address(), 3); err != nil {
		t.Fatalf("Send: %v", err)
	}
	it := bc.Iterator()
	for {
		b, err := it.Next()
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if b == nil {
			break
		}
		for _, tr := range b.Transactions {
			ok, err := bc.VerifyTransaction(tr)
			if err != nil {
				t.Fatalf("VerifyTransaction: %v", err)
			}
			if !ok {
				t.Errorf("transaction %x failed verification", tr.ID)
			}
		}
	}
}

func TestMineBlockRejectsTamperedSignature(t *testing.T) {
	bc, alice := setup(t)
	bob, _ := wallet.NewWallet()

	tr, err := bc.NewUTXOTransaction(alice, bob.Address(), 4)
	if err != nil {
		t.Fatalf("NewUTXOTransaction: %v", err)
	}
	tr.Vin[0].Signature[0] ^= 0xFF // tamper

	if err := bc.MineBlock([]*tx.Transaction{tr}); !errors.Is(err, ErrInvalidTransaction) {
		t.Errorf("MineBlock with tampered signature: err = %v, want ErrInvalidTransaction", err)
	}
}

func TestSendInsufficientFunds(t *testing.T) {
	bc, alice := setup(t)
	bob, _ := wallet.NewWallet()
	if err := bc.Send(alice, bob.Address(), 999); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("err = %v, want ErrInsufficientFunds", err)
	}
}
