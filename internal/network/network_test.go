package network

import (
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/chain"
	"github.com/thefcan/gochain/internal/wallet"
)

func TestSyncReplicatesChainOverTCP(t *testing.T) {
	owner, _ := wallet.NewWallet()
	server, err := chain.CreateBlockchain(filepath.Join(t.TempDir(), "server.db"), owner.Address())
	if err != nil {
		t.Fatalf("CreateBlockchain: %v", err)
	}
	defer server.Close()

	bob, _ := wallet.NewWallet()
	if err := server.Send(owner, bob.Address(), 4); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Serve the chain on a random local port.
	ln, err := NewNode(server).Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	// A fresh, empty node syncs from the server over TCP.
	client, err := chain.OpenOrInit(filepath.Join(t.TempDir(), "client.db"))
	if err != nil {
		t.Fatalf("OpenOrInit: %v", err)
	}
	defer client.Close()
	cli := NewNode(client)

	added, err := cli.SyncFrom(ln.Addr().String())
	if err != nil {
		t.Fatalf("SyncFrom: %v", err)
	}

	srvHashes, _ := server.GetBlockHashes()
	if added != len(srvHashes) {
		t.Errorf("synced %d blocks, want %d", added, len(srvHashes))
	}
	if bal, _ := client.Balance(owner.Address()); bal != 6 {
		t.Errorf("client owner balance = %d, want 6", bal)
	}
	if bal, _ := client.Balance(bob.Address()); bal != 4 {
		t.Errorf("client bob balance = %d, want 4", bal)
	}

	// Re-syncing is idempotent.
	again, err := cli.SyncFrom(ln.Addr().String())
	if err != nil {
		t.Fatalf("re-sync: %v", err)
	}
	if again != 0 {
		t.Errorf("re-sync added %d blocks, want 0", again)
	}
}
