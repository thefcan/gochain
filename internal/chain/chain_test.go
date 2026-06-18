package chain

import (
	"path/filepath"
	"testing"

	"github.com/thefcan/gochain/internal/pow"
)

func tempChain(t *testing.T) *Blockchain {
	t.Helper()
	bc, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { bc.Close() })
	return bc
}

func TestOpenCreatesGenesis(t *testing.T) {
	bc := tempChain(t)
	b, err := bc.Iterator().Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if b == nil {
		t.Fatal("empty chain; expected a genesis block")
	}
	if string(b.Data) != "Genesis Block" {
		t.Errorf("first block data = %q, want genesis", b.Data)
	}
}

func TestAddBlockAndIterate(t *testing.T) {
	bc := tempChain(t)
	for _, d := range []string{"first", "second"} {
		if err := bc.AddBlock(d); err != nil {
			t.Fatalf("AddBlock(%q): %v", d, err)
		}
	}

	var datas []string
	it := bc.Iterator()
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
		datas = append(datas, string(b.Data))
	}

	want := []string{"second", "first", "Genesis Block"} // tip -> genesis
	if len(datas) != len(want) {
		t.Fatalf("chain length = %d, want %d", len(datas), len(want))
	}
	for i := range want {
		if datas[i] != want[i] {
			t.Errorf("block %d = %q, want %q", i, datas[i], want[i])
		}
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")

	bc, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := bc.AddBlock("durable"); err != nil {
		t.Fatalf("AddBlock: %v", err)
	}
	bc.Close()

	// Reopen the same file in a new Blockchain: the data must survive.
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	b, err := reopened.Iterator().Next()
	if err != nil {
		t.Fatalf("Next after reopen: %v", err)
	}
	if b == nil || string(b.Data) != "durable" {
		t.Errorf("tip after reopen = %v, want data %q", b, "durable")
	}
}
